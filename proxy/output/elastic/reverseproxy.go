// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package elastic

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"sort"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/core/stats"
	task2 "infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/proxy/balancer"
)

type ReverseProxy struct {
	oldAddr                  string
	bla                      balancer.IBalancer
	proxyConfig              *ProxyConfig
	endpoints                []string
	lastNodesTopologyVersion int

	hostClients map[string]*fasthttp.HostClient
	clients     map[string]*fasthttp.Client
	locker      sync.RWMutex

	fixedClient bool
	client      fasthttp.ClientAPI
	host        string
	HTTPPool    *fasthttp.RequestResponsePool
}

func isEndpointValid(node elastic.NodesInfo, cfg *ProxyConfig) bool {

	var hasExclude = false
	var hasInclude = false
	endpoint := node.GetHttpPublishHost()

	if global.Env().IsDebug {
		log.Tracef("validate endpoint %v", endpoint)
	}

	for _, v := range cfg.Filter.Hosts.Exclude {
		hasExclude = true
		if endpoint == v {
			log.Debugf("host [%v] in exclude list, mark as invalid", endpoint)
			return false
		}
	}

	for _, v := range cfg.Filter.Hosts.Include {
		hasInclude = true
		if endpoint == v {
			log.Debugf("host [%v] in include list, mark as valid", endpoint)
			return true
		}
	}

	//no exclude and only have include, means white list mode
	if !hasExclude && hasInclude {
		return false
	}

	hasExclude = false
	hasInclude = false
	for _, v := range cfg.Filter.Roles.Exclude {
		hasExclude = true
		if util.ContainsAnyInArray(v, node.Roles) {
			log.Debugf("node [%v] role [%v] match exclude rule [%v], mark as invalid", endpoint, node.Roles, v)
			return false
		}
	}

	for _, v := range cfg.Filter.Roles.Include {
		hasInclude = true
		if util.ContainsAnyInArray(v, node.Roles) {
			log.Debugf("node [%v] role [%v] match include rule [%v], mark as valid", endpoint, node.Roles, v)
			return true
		}
	}

	if !hasExclude && hasInclude {
		return false
	}

	hasExclude = false
	hasInclude = false
	for _, o := range cfg.Filter.Tags.Exclude {
		hasExclude = true
		for k, v := range o {
			v1, ok := node.Attributes[k]
			if ok {
				if v1 == v {
					log.Debugf("node [%v] tags [%v:%v] in exclude list, mark as invalid", endpoint, k, v)
					return false
				}
			}
		}
	}

	for _, o := range cfg.Filter.Tags.Include {
		hasInclude = true
		for k, v := range o {
			v1, ok := node.Attributes[k]
			if ok {
				if v1 == v {
					log.Debugf("node [%v] tags [%v:%v] in include list, mark as valid", endpoint, k, v)
					return true
				}
			}
		}
	}

	if !hasExclude && hasInclude {
		return false
	}

	return true
}

func (p *ReverseProxy) refreshNodes(force bool) {

	p.locker.Lock()
	defer p.locker.Unlock()

	if global.Env().IsDebug {
		log.Trace("elasticsearch client nodes refreshing")
	}
	cfg := p.proxyConfig

	ws := []int{}
	esConfig := elastic.GetConfig(cfg.Elasticsearch)

	metadata := elastic.GetOrInitMetadata(esConfig)

	if metadata == nil && !force {
		log.Trace("metadata is nil and not forced, skip nodes refresh")
		return
	}

	if !esConfig.Discovery.Enabled && !force {
		log.Trace("discovery is not enabled, skip nodes refresh")
		return
	}

	hosts := []string{}
	checkMetadata := false
	if metadata != nil && metadata.Nodes != nil && len(*metadata.Nodes) > 0 {

		oldV := p.lastNodesTopologyVersion
		p.lastNodesTopologyVersion = metadata.NodesTopologyVersion

		if oldV == p.lastNodesTopologyVersion {
			if global.Env().IsDebug {
				log.Trace("metadata.NodesTopologyVersion is equal")
			}
			return
		}

		checkMetadata = true
		for _, y := range *metadata.Nodes {
			if !isEndpointValid(y, cfg) {
				continue
			}

			host := y.GetHttpPublishHost()
			if host != "" && elastic.IsHostAvailable(host) {
				hosts = append(hosts, host)
			}
		}
		log.Tracef("discovery %v nodes: [%v]", len(hosts), util.JoinArray(hosts, ", "))
	}

	if len(hosts) == 0 {
		hosts = metadata.GetSeedHosts()
		if checkMetadata {
			log.Debugf("no matched endpoint, fallback to seed: %v", hosts)
		}
	}

	newHosts := []string{}
	for _, endpoint := range hosts {
		if !elastic.IsHostAvailable(endpoint) {
			log.Debug(endpoint, " is not available")
			continue
		}
		_, ok := p.hostClients[endpoint]
		if !ok {
			p.hostClients[endpoint] = &fasthttp.HostClient{
				Name:                          "reverse_proxy",
				Addr:                          endpoint,
				DisableHeaderNamesNormalizing: true,
				DisablePathNormalizing:        true,
				MaxConns:                      cfg.MaxConnection,
				MaxResponseBodySize:           cfg.MaxResponseBodySize,
				MaxConnWaitTimeout:            cfg.MaxConnWaitTimeout,
				MaxConnDuration:               cfg.MaxConnDuration,
				MaxIdleConnDuration:           cfg.MaxIdleConnDuration,
				ReadTimeout:                   cfg.ReadTimeout,
				WriteTimeout:                  cfg.WriteTimeout,
				ReadBufferSize:                cfg.ReadBufferSize,
				WriteBufferSize:               cfg.WriteBufferSize,
				//RetryIf: func(request *fasthttp.Request) bool {
				//
				//},
				Dial: func(addr string) (net.Conn, error) {
					return fasthttp.DialTimeout(addr, cfg.DialTimeout)
				},
				IsTLS: metadata.IsTLS(),
				TLSConfig: &tls.Config{
					InsecureSkipVerify: cfg.TLSInsecureSkipVerify,
				},
			}
		}

		_, ok = p.clients[endpoint]
		if !ok {
			p.clients[endpoint] = &fasthttp.Client{
				Name:                          "reverse_proxy",
				DisableHeaderNamesNormalizing: true,
				DisablePathNormalizing:        true,
				MaxConnsPerHost:               cfg.MaxConnection,
				MaxResponseBodySize:           cfg.MaxResponseBodySize,
				MaxConnWaitTimeout:            cfg.MaxConnWaitTimeout,
				MaxConnDuration:               cfg.MaxConnDuration,
				MaxIdleConnDuration:           cfg.MaxIdleConnDuration,
				ReadTimeout:                   cfg.ReadTimeout,
				WriteTimeout:                  cfg.WriteTimeout,
				ReadBufferSize:                cfg.ReadBufferSize,
				WriteBufferSize:               cfg.WriteBufferSize,
				Dial: func(addr string) (net.Conn, error) {
					return fasthttp.DialDualStackTimeout(addr, cfg.DialTimeout)
				},
				TLSConfig: &tls.Config{
					InsecureSkipVerify: cfg.TLSInsecureSkipVerify,
				},
			}
		}

		//get predefined weights
		w, o := cfg.Weights[endpoint]
		if !o || w <= 0 {
			w = 1
		}
		ws = append(ws, w)
		newHosts = append(newHosts, endpoint)
	}

	if len(newHosts) == 0 {
		log.Errorf("upstream for [%v] is empty", esConfig.Name)
		return
	}

	sort.Strings(newHosts)

	if util.JoinArray(newHosts, ", ") == util.JoinArray(p.endpoints, ", ") {
		log.Debugf("hosts of [%v] no change, skip", esConfig.Name)
		return
	}

	//replace with new hostClients
	//TODO add locker
	p.bla = balancer.NewBalancer(ws)
	newHostsStr := util.JoinArray(newHosts, ", ")
	if rate.GetRateLimiterPerSecond("elasticsearch", esConfig.Name+newHostsStr, 1).Allow() {
		log.Infof("elasticsearch [%v] hosts: [%v] => [%v]", esConfig.Name, util.JoinArray(p.endpoints, ", "), newHostsStr)
	}
	p.endpoints = newHosts
	log.Trace(esConfig.Name, " elasticsearch client nodes refreshed")

}

func NewReverseProxy(cfg *ProxyConfig) *ReverseProxy {

	p := ReverseProxy{
		oldAddr:     "",
		proxyConfig: cfg,
		hostClients: map[string]*fasthttp.HostClient{},
		clients:     map[string]*fasthttp.Client{},
		locker:      sync.RWMutex{},
	}

	p.refreshNodes(true)

	if p.proxyConfig.FixedClient {
		if p.proxyConfig.ClientMode == "client" {
			_, p.client, p.host = p.getClient()
		} else {
			_, p.client, p.host = p.getHostClient()
		}
	} else {
		if cfg.Refresh.Enabled {
			log.Debugf("refresh enabled for elasticsearch: [%v]", cfg.Elasticsearch)
			task := task2.ScheduleTask{
				Description: fmt.Sprintf("refresh nodes for elasticsearch [%v]", cfg.Elasticsearch),
				Type:        "interval",
				Interval:    cfg.Refresh.Interval,
				Task: func(ctx context.Context) {
					p.refreshNodes(false)
				},
			}
			task2.RegisterScheduleTask(task)
		}
	}

	p.HTTPPool = fasthttp.NewRequestResponsePool("es_proxy_" + cfg.Elasticsearch)

	return &p
}

func (p *ReverseProxy) getHostClient() (clientAvailable bool, client *fasthttp.HostClient, endpoint string) {
	if p.hostClients == nil {
		panic("ReverseProxy has been closed")
	}

	if len(p.hostClients) == 0 || len(p.endpoints) == 0 {
		log.Error("no upstream found")
		return false, nil, ""
	}

	if p.bla != nil {
		// bla has been opened
		idx := p.bla.Distribute()
		if idx >= len(p.endpoints) {
			log.Warn("invalid offset, ", idx, " vs ", len(p.hostClients), p.endpoints, ", random pick now")
			idx = 0
			goto RANDOM
		}

		// if len(p.bla.) != len(p.endpoints) {
		// 	log.Warn("hostClients != endpoints, ", len(hostClients), " vs ", len(p.endpoints), ", random pick now")
		// 	goto RANDOM
		// }

		e := p.endpoints[idx]
		c, ok := p.hostClients[e] //TODO, check client by endpoint
		if !ok {
			log.Error("client not found for: ", e)
		}

		return true, c, e
	}

RANDOM:
	//or go random way
	max := len(p.hostClients)
	seed := rand.Intn(max)
	if seed >= len(p.hostClients) || seed >= len(p.endpoints) {
		log.Warn("invalid upstream offset, reset to 0")
		seed = 0
	}
	e := p.endpoints[seed]
	c := p.hostClients[e]
	return true, c, e
}

func (p *ReverseProxy) getClient() (clientAvailable bool, client *fasthttp.Client, endpoint string) {

	if p.clients == nil {
		panic("ReverseProxy has been closed")
	}

	if len(p.clients) == 0 || len(p.endpoints) == 0 {
		p.refreshNodes(true)
		if p.clients == nil || len(p.clients) == 0 || len(p.endpoints) == 0 {
			panic("no upstream found")
		}
	}

	p.locker.RLock()
	defer p.locker.RUnlock()

	if p.bla != nil {
		// bla has been opened
		idx := p.bla.Distribute()
		if idx >= len(p.endpoints) {
			log.Warn("invalid offset, ", idx, " vs ", len(p.clients), p.endpoints, ", random pick now")
			idx = 0
			goto RANDOM
		}

		e := p.endpoints[idx]
		c, ok := p.clients[e] //TODO, check client by endpoint
		if !ok {
			log.Error("client not found for: ", e)
		}

		return true, c, e
	}

RANDOM:
	//or go random way
	max := len(p.clients)
	seed := rand.Intn(max)
	if seed >= len(p.clients) || seed >= len(p.endpoints) {
		log.Warn("invalid upstream offset, reset to 0")
		seed = 0
	}
	e := p.endpoints[seed]
	c := p.clients[e]
	return true, c, e
}

var failureMessage = []string{"connection refused", "no such host", "timed out", "Connection: close"}

func (p *ReverseProxy) DelegateRequest(elasticsearch string, metadata *elastic.ElasticsearchMetadata, myctx *fasthttp.RequestCtx) {

	if p.proxyConfig.SkipEnrichMetadata {
		//update context
		if myctx.Has("elastic_cluster_name") {
			es1 := myctx.MustGetStringArray("elastic_cluster_name")
			myctx.Set("elastic_cluster_name", append(es1, elasticsearch))
		} else {
			myctx.Set("elastic_cluster_name", []string{elasticsearch})
		}
	}

	res := p.HTTPPool.AcquireResponseWithTag("proxy_response")
	defer p.HTTPPool.ReleaseResponse(res)

	if !p.proxyConfig.SkipCleanupHopHeaders {
		cleanHopHeaders(&myctx.Request)
	}

	var pc fasthttp.ClientAPI
	var host string

	if p.proxyConfig.FixedClient {
		pc = p.client
		host = p.host
	} else {
		//var ok bool
		//使用算法来获取合适的 client
		switch metadata.Config.ClientMode {
		case "client":
			_, pc, host = p.getClient()
			break
		case "host":
			_, pc, host = p.getHostClient()
			break
		default:
			_, pc, host = p.getClient()
		}

		if !p.proxyConfig.SkipAvailableCheck && !elastic.IsHostAvailable(host) {
			old := host
			host = metadata.GetActiveHost()
			if rate.GetRateLimiterPerSecond("proxy-host-not-available", old, 1).Allow() {
				log.Infof("host [%v] is not available, fallback: [%v]", old, host)
			}
			pc = metadata.GetHttpClient(host)
		}
	}

	// modify schema，align with elasticsearch's schema
	originalHost := string(myctx.Request.Header.Host())
	originalSchema := myctx.Request.GetSchema()
	schemaChanged := false
	clonedURI := myctx.Request.CloneURI()
	defer fasthttp.ReleaseURI(clonedURI)
	if metadata.GetSchema() != originalSchema {
		if !p.proxyConfig.SkipEnrichMetadata {
			myctx.Request.Header.Set("X-Forwarded-Proto", originalSchema)
		}

		clonedURI.SetScheme(metadata.GetSchema())
		myctx.Request.SetURI(clonedURI)
		_, pc, host = p.getClient()

		if !p.proxyConfig.SkipAvailableCheck && !elastic.IsHostAvailable(host) {
			old := host
			host = metadata.GetActiveHost()
			log.Infof("host [%v] is not available, re-choose one: [%v]", old, host)
			pc = metadata.GetHttpClient(host)
		}

		schemaChanged = true
	}

	if !p.proxyConfig.SkipEnrichMetadata {
		forwardedFor := myctx.Request.Header.Peek(fasthttp.HeaderXForwardedFor)
		remoteAddr := myctx.RemoteAddr().String()
		if len(forwardedFor) == 0 {
			myctx.Request.Header.Set(fasthttp.HeaderXForwardedFor, remoteAddr)
		} else {
			myctx.Request.Header.Set(fasthttp.HeaderXForwardedFor, string(forwardedFor)+", "+remoteAddr)
		}
		myctx.Request.Header.Add(fasthttp.HeaderXRealIP, myctx.RemoteAddr().String())
		myctx.Request.Header.Add(fasthttp.HeaderXForwardedHost, originalHost)
	}

	if global.Env().IsDebug {
		log.Tracef("send request [%v] to upstream [%v]", myctx.Request.PhantomURI().String(), host)
	}

	curHost := string(myctx.Request.Host())
	if host != curHost || host != originalHost {
		myctx.Request.SetHostBytes([]byte(host))
	}

	retry := 0
START:

	metadata.CheckNodeTrafficThrottle(host, 1, myctx.Request.GetRequestLength(), 0)

	//if p.proxyConfig.Timeout <= 0 {
	//	p.proxyConfig.Timeout = 60 * time.Second
	//}

	var err error
	if p.proxyConfig.Timeout > 0 {
		err = pc.DoTimeout(&myctx.Request, res, p.proxyConfig.Timeout)
	} else {
		err = pc.Do(&myctx.Request, res)
	}

	if err != nil {

		retryAble := false

		if util.ContainsAnyInArray(err.Error(), failureMessage) {
			stats.Increment("reverse_proxy", "backend_failure")
			//record translog, update failure ticket
			if global.Env().IsDebug {
				if rate.GetRateLimiterPerSecond(metadata.Config.ID, host+"backend_failure_on_error", 1).Allow() {
					log.Errorf("elasticsearch [%v][%v] is on fire now, %v", p.proxyConfig.Elasticsearch, host, err)
					time.Sleep(1 * time.Second)
				}
			}
			if !p.proxyConfig.SkipAvailableCheck {
				elastic.GetOrInitHost(host, metadata.Config.ID).ReportFailure()
			}
			//server failure flow
		} else if res.StatusCode() == 429 {
			if p.proxyConfig.RetryOnBackendBusy {
				retryAble = true
			}
		}

		if retryAble {
			retry++
			if p.proxyConfig.MaxRetryTimes > 0 && retry < p.proxyConfig.MaxRetryTimes {
				if p.proxyConfig.RetryDelayInMs > 0 {
					time.Sleep(time.Duration(p.proxyConfig.RetryDelayInMs) * time.Millisecond)
				}
				myctx.Request.Header.Add("RETRY_AT", time.Now().String())
				goto START
			} else {
				log.Debugf("reached max retries, failed to proxy request: %v, %v", err, string(myctx.Request.Header.RequestURI()))
			}
		} else {
			if rate.GetRateLimiterPerSecond(metadata.Config.ID, host+"backend_failure_on_error", 1).Allow() {
				log.Warnf("failed to proxy request: %v to host %v, %v, retried: #%v, error:%v", string(myctx.Request.Header.RequestURI()), host, retry, retry, err)
			}
		}

		//TODO if backend failure and after reached max retry, should save translog and mark the elasticsearch cluster to downtime, deny any new requests
		// the translog file should consider to contain dirty writes, could be used to do cross cluster check or manually operations recovery.

		res.Header.SetContentType(util.ContentTypeJson)
		res.SwapBody([]byte(fmt.Sprintf("{\"error\":true,\"message\":\"%v\"}", err.Error())))
		res.SetStatusCode(500)
	} else {
		if global.Env().IsDebug {
			log.Tracef("request [%v] [%v] [%v] [%v]", myctx.Request.PhantomURI().String(), util.SubString(util.UnsafeBytesToString(myctx.Request.GetRawBody()), 0, 256), res.StatusCode(), util.SubString(util.UnsafeBytesToString(res.GetRawBody()), 0, 256))
		}
	}

	if !p.proxyConfig.SkipKeepOriginalURI {
		// restore schema
		if schemaChanged {
			clonedURI.SetScheme(originalSchema)
			myctx.Request.SetURI(clonedURI)
		}

		if host != originalHost && originalHost != "" {
			myctx.Request.SetHost(originalHost)
		}
	}

	//update
	if !p.proxyConfig.SkipEnrichMetadata {

		if retry > 0 {
			res.Header.Set("X-Retry-Times", util.ToString(retry))
		}

		res.Header.Set("X-Backend-Cluster", p.proxyConfig.Elasticsearch)
		res.Header.Set("X-Backend-Server", host)
		myctx.SetDestination(host)
	}

	//merge response
	myctx.Response.CopyMergeHeader(res)

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
