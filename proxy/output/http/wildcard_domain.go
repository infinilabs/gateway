/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package http

import (
	"fmt"
	"infini.sh/framework/core/api"
	"infini.sh/framework/lib/fasttemplate"
	"io"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
)

type WildcardDomainFilter struct {
	requestTimeout time.Duration

	Schema          string `config:"schema"`
	SkipFailureHost bool   `config:"skip_failure_host"`
	Suffix          string `config:"suffix"`
	Domain            string `config:"domain"`

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

	suffixTemplate            *fasttemplate.Template

	MaxRedirectsCount int  `config:"max_redirects_count"`
	FollowRedirects   bool `config:"follow_redirects"`
	HTTPPool          *fasthttp.RequestResponsePool
	client            *fasthttp.Client
}

const wildCardHTTFilter = "wildcard_domain"

func (filter *WildcardDomainFilter) Name() string {
	return wildCardHTTFilter
}

func (filter *WildcardDomainFilter) Filter(ctx *fasthttp.RequestCtx) {
	var err error

	var suffix string
	if filter.suffixTemplate != nil {
		suffix = filter.suffixTemplate.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
			variable, err := ctx.GetValue(tag)
			if err == nil {
				return w.Write([]byte(util.ToString(variable)))
			}
			return -1, err
		})
	}

	if suffix == "" {
		ctx.Error("invalid suffix", 400)
		return
	}

	host := suffix+"."+filter.Domain
	err = filter.forward(host, ctx)
	if err != nil {
		ctx.Response.SetBodyString(err.Error())
		return
	}
}

func (filter *WildcardDomainFilter) forward(host string, ctx *fasthttp.RequestCtx) (err error) {

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

	if filter.FollowRedirects {
		err = filter.client.DoRedirects(&ctx.Request, res, filter.MaxRedirectsCount)
	} else {
		if filter.requestTimeout > 0 {
			err = filter.client.DoTimeout(&ctx.Request, res, filter.requestTimeout)
		} else {
			err = filter.client.Do(&ctx.Request, res)
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

	ctx.Response.Header.Set("X-Backend-Server", host)

	return err
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata(wildCardHTTFilter, NewWildcardHTTPFilter, &WildcardDomainFilter{})
}

func NewWildcardHTTPFilter(c *config.Config) (pipeline.Filter, error) {

	runner := WildcardDomainFilter{
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

	runner.client = &fasthttp.Client{
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
	if strings.Contains(runner.Suffix, "$[[") {
		var err error
		runner.suffixTemplate, err = fasttemplate.NewTemplate(runner.Suffix, "$[[", "]]")
		if err != nil {
			panic(err)
		}
	}
	runner.HTTPPool = fasthttp.NewRequestResponsePool(wildCardHTTFilter)

	return &runner, nil
}
