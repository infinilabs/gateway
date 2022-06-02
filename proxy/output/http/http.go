/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package http

import (
	"crypto/tls"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"math/rand"
	"sync"
	"time"
)

type HTTPFilter struct {
	requestTimeout time.Duration

	MaxConnsPerHost int      `config:"max_connection_per_host"`
	Schema          string   `config:"schema"`
	SkipFailureHost bool     `config:"skip_failure_host"`
	Host            string   `config:"host"`
	Hosts           []string `config:"hosts"`
	//client          *fasthttp.Client
	clients sync.Map //*fasthttp.HostClient

	//host
	MaxConnection       int `config:"max_connection_per_node"`
	MaxResponseBodySize int `config:"max_response_size"`
	MaxRetryTimes       int `config:"max_retry_times"`
	RetryDelayInMs      int `config:"retry_delay_in_ms"`

	MaxConnWaitTimeout    time.Duration `config:"max_conn_wait_timeout"`
	MaxIdleConnDuration   time.Duration `config:"max_idle_conn_duration"`
	MaxConnDuration       time.Duration `config:"max_conn_duration"`
	Timeout               time.Duration `config:"timeout"`
	ReadTimeout           time.Duration `config:"read_timeout"`
	WriteTimeout          time.Duration `config:"write_timeout"`
	ReadBufferSize        int           `config:"read_buffer_size"`
	WriteBufferSize       int           `config:"write_buffer_size"`
	TLSInsecureSkipVerify bool          `config:"tls_insecure_skip_verify"`
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
				return
			}
		}
		if err != nil {
			ctx.Response.SetBodyString(err.Error())
			return
		}
	}

}

func (filter *HTTPFilter) forward(host string, ctx *fasthttp.RequestCtx) (err error) {
	orignalHost := string(ctx.Request.URI().Host())
	orignalSchema := string(ctx.Request.URI().Scheme())

	ctx.URI().SetHost(host)
	ctx.Request.SetHost(host)

	//keep original host
	ctx.Request.Header.SetHost(orignalHost)

	ctx.Request.Header.Add("X-Forwarded-For", ctx.RemoteAddr().String())
	ctx.Request.Header.Add("X-Real-IP", ctx.RemoteAddr().String())
	ctx.Request.Header.Add("X-Forwarded-Host", orignalHost)

	ctx.Request.URI().SetScheme(filter.Schema)

	if global.Env().IsDebug {
		log.Debug("forward http request:", ctx.URI().String(), ctx.Request.String())
	}
	c, ok := filter.clients.Load(host)
	if ok {
		client, ok := c.(*fasthttp.HostClient)
		if !ok {
			return errors.Errorf("invalid host client:", host)
		}

		if filter.requestTimeout > 0 {
			err = client.DoTimeout(&ctx.Request, &ctx.Response, filter.requestTimeout)
		} else {
			err = client.Do(&ctx.Request, &ctx.Response)
		}

		ctx.Request.URI().SetScheme(orignalSchema)
		ctx.Request.SetHost(orignalHost)

		ctx.Response.Header.Set("X-Backend-Server", host)
	} else {
		return errors.Errorf("invalid host client:", host)
	}
	return err
}

func init() {
	pipeline.RegisterFilterPluginWithConfigMetadata("http", NewHTTPFilter, &HTTPFilter{})
}

func NewHTTPFilter(c *config.Config) (pipeline.Filter, error) {

	runner := HTTPFilter{
		MaxConnsPerHost:       10000,
		SkipFailureHost:       true,
		MaxConnection:         5000,
		MaxRetryTimes:         0,
		RetryDelayInMs:        1000,
		TLSInsecureSkipVerify: true,
		ReadBufferSize:        4096 * 4,
		WriteBufferSize:       4096 * 4,
		//maxt wait timeout for free connection
		MaxConnWaitTimeout: util.GetDurationOrDefault("30s", 30*time.Second),

		//keep alived connection
		MaxConnDuration: util.GetDurationOrDefault("0s", 0*time.Second),

		ReadTimeout:  util.GetDurationOrDefault("0s", 0*time.Hour),
		Timeout:      util.GetDurationOrDefault("30s", 30*time.Second),
		WriteTimeout: util.GetDurationOrDefault("0s", 0*time.Hour),
		//idle alive connection will be closed
		MaxIdleConnDuration: util.GetDurationOrDefault("30s", 30*time.Second),
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
		c := &fasthttp.HostClient{
			Name:                          "reverse_proxy",
			Addr:                          host,
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
			IsTLS:                         runner.Schema == "https",
			MaxConns:                      runner.MaxConnection,
			MaxResponseBodySize:           runner.MaxResponseBodySize,
			MaxConnWaitTimeout:            runner.MaxConnWaitTimeout,
			MaxConnDuration:               runner.MaxConnDuration,
			MaxIdleConnDuration:           runner.MaxIdleConnDuration,
			ReadTimeout:                   runner.ReadTimeout,
			WriteTimeout:                  runner.WriteTimeout,
			ReadBufferSize:                runner.ReadBufferSize,
			WriteBufferSize:               runner.WriteBufferSize,
			TLSConfig: &tls.Config{
				InsecureSkipVerify: runner.TLSInsecureSkipVerify,
			},
		}
		runner.clients.Store(host, c)
	}

	return &runner, nil
}
