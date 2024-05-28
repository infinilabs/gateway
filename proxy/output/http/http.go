/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package http

import (
	"fmt"
	"infini.sh/framework/core/api"
	"math/rand"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type HTTPFilter struct {
	requestTimeout time.Duration

	Schema          string   `config:"schema"`
	SkipFailureHost bool     `config:"skip_failure_host"`
	Host            string   `config:"host"`
	Hosts           []string `config:"hosts"`
	clients         sync.Map //*fasthttp.Client

	//host
	MaxConnection       int `config:"max_connection_per_node"`
	MaxResponseBodySize int `config:"max_response_size"`
	MaxRetryTimes       int `config:"max_retry_times"`
	RetryDelayInMs      int `config:"retry_delay_in_ms"`

	SkipCleanupHopHeaders bool `config:"skip_cleanup_hop_headers"`
	SkipEnrichMetadata    bool `config:"skip_metadata_enrich"`

	MaxConnWaitTimeout    time.Duration `config:"max_conn_wait_timeout"`
	MaxIdleConnDuration   time.Duration `config:"max_idle_conn_duration"`
	MaxConnDuration       time.Duration `config:"max_conn_duration"`
	Timeout               time.Duration `config:"timeout"`
	ReadTimeout           time.Duration `config:"read_timeout"`
	WriteTimeout          time.Duration `config:"write_timeout"`
	ReadBufferSize        int           `config:"read_buffer_size"`
	WriteBufferSize       int           `config:"write_buffer_size"`
	TLSInsecureSkipVerify bool          `config:"tls_insecure_skip_verify"`

	TLSConfig *config.TLSConfig `config:"tls"` //client tls config

	MaxRedirectsCount int  `config:"max_redirects_count"`
	FollowRedirects   bool `config:"follow_redirects"`
	HTTPPool          *fasthttp.RequestResponsePool
}

func (filter *HTTPFilter) Name() string {
	return "http"
}

func (filter *HTTPFilter) getHost() string {
	max := len(filter.Hosts)
	if max == 1 {
		return filter.Hosts[0]
	}

	seed := rand.Intn(max)
	if seed >= len(filter.Hosts) {
		log.Warn("invalid upstream offset, reset to 0")
		seed = 0
	}
	return filter.Hosts[seed]
}

func (filter *HTTPFilter) Filter(ctx *fasthttp.RequestCtx) {
	var err error

	host := filter.getHost()
	err = filter.forward(host, ctx)
	if err == nil {
		return
	}

	if filter.SkipFailureHost {
		for _, v := range filter.Hosts {
			err = filter.forward(v, ctx)
			if err == nil {
				break
			}
		}
		if err != nil {
			ctx.Response.SetBodyString(err.Error())
			return
		}
	}

}

// Hop-by-hop headers. These are removed when sent to the backend.
// As of RFC 7230, hop-by-hop headers are required to appear in the
// Connection header field. These are the headers defined by the
// obsoleted RFC 2616 (section 13.5.1) and are used for backward
// compatibility.
var hopHeaders = []string{
	"Connection",          // Connection
	"Proxy-Connection",    // non-standard but still sent by libcurl and rejected by e.g. google
	"Keep-Alive",          // Keep-Alive
	"Proxy-Authenticate",  // Proxy-Authenticate
	"Proxy-Authorization", // Proxy-Authorization
	"Te",                  // canonicalized version of "TE"
	"Trailer",             // not Trailers per URL above; https://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",   // Transfer-Encoding
	"Upgrade",             // Upgrade

	//"Accept-Encoding",             // Disable Gzip
	//"Content-Encoding",             // Disable Gzip
}

func cleanHopHeaders(req *fasthttp.Request) {
	for _, h := range hopHeaders {
		req.Header.Del(h)
	}
}

func (filter *HTTPFilter) forward(host string, ctx *fasthttp.RequestCtx) (err error) {

	if !filter.SkipCleanupHopHeaders {
		cleanHopHeaders(&ctx.Request)
	}

	orignalHost := string(ctx.Request.PhantomURI().Host())
	orignalSchema := string(ctx.Request.PhantomURI().Scheme())

	if host == "" {
		panic("invalid host")
	}

	ctx.Request.SetHost(host)

	//keep original host
	ctx.Request.Header.SetHost(orignalHost)

	if !filter.SkipEnrichMetadata {
		ctx.Request.Header.Set(fasthttp.HeaderXForwardedFor, ctx.RemoteAddr().String())
		ctx.Request.Header.Set(fasthttp.HeaderXRealIP, ctx.RemoteAddr().String())
		ctx.Request.Header.Set(fasthttp.HeaderXForwardedHost, orignalHost)
	}

	clonedURI := ctx.Request.CloneURI()
	defer fasthttp.ReleaseURI(clonedURI)

	res := filter.HTTPPool.AcquireResponseWithTag("http_response")
	defer filter.HTTPPool.ReleaseResponse(res)

	clonedURI.SetScheme(filter.Schema)
	ctx.Request.SetURI(clonedURI)

	if global.Env().IsDebug {
		log.Tracef("forward http request: %v, %v", ctx.PhantomURI().String(), ctx.Request.String())
	}

	c, ok := filter.clients.Load(host)
	if ok {
		client, ok := c.(*fasthttp.Client)
		if !ok {
			return errors.Errorf("invalid host client:", host)
		}

		if filter.FollowRedirects {
			err = client.DoRedirects(&ctx.Request, res, filter.MaxRedirectsCount)
		} else {
			if filter.requestTimeout > 0 {
				err = client.DoTimeout(&ctx.Request, res, filter.requestTimeout)
			} else {
				err = client.Do(&ctx.Request, res)
			}
		}

		clonedURI.SetScheme(orignalSchema)
		ctx.Request.SetURI(clonedURI)
		ctx.Request.SetHost(orignalHost)

		//merge response
		ctx.Response.CopyMergeHeader(res)

		if err != nil {
			log.Error(err, string(ctx.Response.GetRawBody()))
		}

	} else {
		err = errors.Errorf("invalid host client:", host)
		log.Warn(err)
	}

	ctx.Response.Header.Set("X-Backend-Server", host)

	return err
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("http", NewHTTPFilter, &HTTPFilter{})
}

func NewHTTPFilter(c *config.Config) (pipeline.Filter, error) {

	runner := HTTPFilter{
		SkipFailureHost:       true,
		MaxConnection:         5000,
		MaxRetryTimes:         0,
		MaxRedirectsCount:     10,
		RetryDelayInMs:        1000,
		TLSInsecureSkipVerify: true,
		ReadBufferSize:        4096 * 4,
		WriteBufferSize:       4096 * 4,
		//max wait timeout for free connection
		MaxConnWaitTimeout: util.GetDurationOrDefault("30s", 30*time.Second),

		//keep alived connection
		MaxConnDuration: util.GetDurationOrDefault("0s", 0*time.Second),

		ReadTimeout:  util.GetDurationOrDefault("0s", 0*time.Second),
		Timeout:      util.GetDurationOrDefault("30s", 0*time.Second),
		WriteTimeout: util.GetDurationOrDefault("0s", 30*time.Second),
		//idle alive connection will be closed
		MaxIdleConnDuration: util.GetDurationOrDefault("300s", 300*time.Second),
	}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	runner.requestTimeout = time.Duration(runner.Timeout) * time.Second

	if runner.Host != "" {
		runner.Hosts = append(runner.Hosts, runner.Host)
	}

	if len(runner.Hosts) <= 0 {
		panic("hosts for http filter can't be nil")
	}

	runner.clients = sync.Map{}

	for _, host := range runner.Hosts {

		c := &fasthttp.Client{
			Name:                          "reverse_proxy",
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
			MaxConnsPerHost:               runner.MaxConnection,
			MaxResponseBodySize:           runner.MaxResponseBodySize,
			MaxConnWaitTimeout:            runner.MaxConnWaitTimeout,
			MaxConnDuration:               runner.MaxConnDuration,
			MaxIdleConnDuration:           runner.MaxIdleConnDuration,
			ReadTimeout:                   runner.ReadTimeout,
			WriteTimeout:                  runner.WriteTimeout,
			ReadBufferSize:                runner.ReadBufferSize,
			WriteBufferSize:               runner.WriteBufferSize,
			DialDualStack:                 true,
			TLSConfig:                     api.SimpleGetTLSConfig(runner.TLSConfig),
		}

		runner.clients.Store(host, c)
	}

	runner.HTTPPool=fasthttp.NewRequestResponsePool("http_filter")


	return &runner, nil
}

