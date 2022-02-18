package elastic

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"strings"
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

	 hostClients  map[string]*fasthttp.HostClient
	 clients  map[string]*fasthttp.Client
	 locker  sync.RWMutex
}

func isEndpointValid(node elastic.NodesInfo, cfg *ProxyConfig) bool {

	var hasExclude = false
	var hasInclude = false
	endpoint := node.GetHttpPublishHost()

	if global.Env().IsDebug{
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

	if !esConfig.Discovery.Enabled && !force{
		log.Trace("discovery is not enabled, skip nodes refresh")
		return
	}


	hosts := []string{}
	checkMetadata := false
	if metadata != nil &&metadata.Nodes!=nil && len(*metadata.Nodes) > 0 {

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

			endpoint:=y.GetHttpPublishHost()
			arry:=strings.Split(endpoint,":")
			if len(arry)==2{
				if !util.TestTCPPort(arry[0],arry[1]){
					log.Debugf("[%v] endpoint [%v] is not available",y.Name,endpoint)
					continue
				}
			}

			hosts = append(hosts, endpoint)
		}
		log.Tracef("discovery %v nodes: [%v]", len(hosts), util.JoinArray(hosts, ", "))
	}

	if len(hosts) == 0 {
		hosts = metadata.GetSeedHosts()
		if checkMetadata {
			log.Debugf("no matched endpoint, fallback to seed: %v", hosts)
		}
	}

	for _, endpoint := range hosts {

		if !elastic.IsHostAvailable(endpoint){
			log.Info(endpoint," is not available")
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
				MaxConnWaitTimeout:  cfg.MaxConnWaitTimeout,
				MaxConnDuration:     cfg.MaxConnDuration,
				MaxIdleConnDuration: cfg.MaxIdleConnDuration,
				ReadTimeout:         cfg.ReadTimeout,
				WriteTimeout:        cfg.WriteTimeout,
				ReadBufferSize:      cfg.ReadBufferSize,
				WriteBufferSize:     cfg.WriteBufferSize,
				//RetryIf: func(request *fasthttp.Request) bool {
				//
				//},
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
				MaxConnsPerHost:                      cfg.MaxConnection,
				MaxResponseBodySize:           cfg.MaxResponseBodySize,
				MaxConnWaitTimeout:  cfg.MaxConnWaitTimeout,
				MaxConnDuration:     cfg.MaxConnDuration,
				MaxIdleConnDuration: cfg.MaxIdleConnDuration,
				ReadTimeout:         cfg.ReadTimeout,
				WriteTimeout:        cfg.WriteTimeout,
				ReadBufferSize:      cfg.ReadBufferSize,
				WriteBufferSize:     cfg.WriteBufferSize,
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
	}

	if len(p.hostClients) == 0 {
		log.Error("proxy upstream is empty")
		return
	}

	if util.JoinArray(hosts, ", ")==util.JoinArray(p.endpoints, ", "){
		log.Debug("hosts no change, skip")
		return
	}

	//replace with new hostClients
	//TODO add locker
	p.bla = balancer.NewBalancer(ws)
	log.Infof("elasticsearch [%v] hosts: [%v] => [%v]", esConfig.Name, util.JoinArray(p.endpoints, ", "), util.JoinArray(hosts, ", "))
	p.endpoints = hosts
	log.Trace(esConfig.Name, " elasticsearch client nodes refreshed")

}

func NewReverseProxy(cfg *ProxyConfig) *ReverseProxy {

	p := ReverseProxy{
		oldAddr:     "",
		proxyConfig: cfg,
		 hostClients : map[string]*fasthttp.HostClient{},
		 clients : map[string]*fasthttp.Client{},
		 locker : sync.RWMutex{},
	}

	p.refreshNodes(true)

	if cfg.Refresh.Enabled {
		log.Debugf("refresh enabled for elasticsearch: [%v]", cfg.Elasticsearch)
		task := task2.ScheduleTask{
			Description: fmt.Sprintf("refresh nodes for elasticsearch [%v]", cfg.Elasticsearch),
			Type:        "interval",
			Interval:    cfg.Refresh.Interval,
			Task: func() {
				p.refreshNodes(false)
			},
		}
		task2.RegisterScheduleTask(task)
	}

	return &p
}

func (p *ReverseProxy) getHostClient() (clientAvailable bool, client *fasthttp.HostClient, endpoint string) {
	if p.hostClients == nil {
		panic("ReverseProxy has been closed")
	}

	if len(p.hostClients) == 0 ||len(p.endpoints)==0{
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
	if seed >= len(p.hostClients)||seed >= len(p.endpoints) {
		log.Warn("invalid upstream offset, reset to 0")
		seed = 0
	}
	e := p.endpoints[seed]
	c := p.hostClients[e]
	return true, c, e
}

func (p *ReverseProxy) getClient() (clientAvailable bool, client *fasthttp.Client, endpoint string) {

	p.locker.RLock()
	defer p.locker.RUnlock()

	if p.clients == nil {
		panic("ReverseProxy has been closed")
	}

	if len(p.clients) == 0 ||len(p.endpoints)==0{
		log.Error("no upstream found")
		return false, nil, ""
	}

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
	if seed >= len(p.clients)||seed >= len(p.endpoints) {
		log.Warn("invalid upstream offset, reset to 0")
		seed = 0
	}
	e := p.endpoints[seed]
	c := p.clients[e]
	return true, c, e
}



var failureMessage = []string{"connection refused", "connection reset", "no such host", "timed out", "Connection: close"}

func (p *ReverseProxy) DelegateRequest(elasticsearch string, metadata *elastic.ElasticsearchMetadata, myctx *fasthttp.RequestCtx) {

	stats.Increment("cache", "strike")


	//update context
	if myctx.Has("elastic_cluster_name") {
		es1 := myctx.MustGetStringArray("elastic_cluster_name")
		myctx.Set("elastic_cluster_name", append(es1, elasticsearch))
	} else {
		myctx.Set("elastic_cluster_name", []string{elasticsearch})
	}


	retry := 0
START:

	req := &myctx.Request
	res := &myctx.Response

	cleanHopHeaders(req)

	var pc fasthttp.ClientAPI
	var ok bool
	var host string
	//使用算法来获取合适的 client
	switch metadata.Config.ClientMode{
	case "client":
		ok, pc, host = p.getClient()
		break
	case "host":
		ok, pc, host = p.getHostClient()
		break
	//case "pipeline":
		//ok, pc, host = p.getHostClient()
		//break
	default:
		ok, pc, host = p.getClient()
	}

	if !ok {
		//TODO no client available, throw error directly
		log.Error("no client available")
		return
	}

	// modify schema，align with elasticsearch's schema
	orignalHost:=string(req.URI().Host())
	orignalSchema:=string(req.URI().Scheme())
	useClient:=false
	if metadata.GetSchema()!=orignalSchema{
		req.Header.Add("X-Forwarded-Proto",orignalSchema)
		req.URI().SetScheme(metadata.GetSchema())
		ok, pc, host = p.getClient()
		res = fasthttp.AcquireResponse()
		useClient=true
	}

	req.Header.Add("X-Forwarded-For",myctx.RemoteAddr().String())
	req.Header.Add("X-Real-IP",myctx.RemoteAddr().String())
	req.Header.Add("X-Forwarded-Host",orignalHost)

	if global.Env().IsDebug {
		log.Tracef("send request [%v] to upstream [%v]", req.URI().String(), host)
	}

	req.SetHost(host)

	metadata.CheckNodeTrafficThrottle(host,1,req.GetRequestLength(),0)

	err := pc.Do(req, res)

	//stats.Increment("reverse_proxy","do")

	//metadata.CheckNodeTrafficThrottle(util.UnsafeBytesToString(req.Header.Host()),0,res.GetResponseLength(),0)


	// restore schema
	req.URI().SetScheme(orignalSchema)
	req.SetHost(orignalHost)


	//update
	myctx.Response.Header.Set("X-Backend-Cluster", p.proxyConfig.Elasticsearch)
	myctx.Response.Header.Set("X-Backend-Server", host)
	myctx.SetDestination(host)

	if  err != nil {
		if util.ContainsAnyInArray(err.Error(), failureMessage) {
			stats.Increment("reverse_proxy","backend_failure")
			//record translog, update failure ticket
			if global.Env().IsDebug {
				if rate.GetRateLimiterPerSecond(metadata.Config.ID, host+"backend_failure_on_error", 1).Allow() {
					log.Errorf("elasticsearch [%v][%v] is on fire now, %v", p.proxyConfig.Elasticsearch, host,err)
					time.Sleep(1 * time.Second)
				}
			}
			elastic.GetOrInitHost(host).ReportFailure()
			//server failure flow
		} else if res.StatusCode() == 429 {
			retry++
			if p.proxyConfig.maxRetryTimes > 0 && retry < p.proxyConfig.maxRetryTimes {
				if p.proxyConfig.retryDelayInMs > 0 {
					time.Sleep(time.Duration(p.proxyConfig.retryDelayInMs) * time.Millisecond)
				}
				stats.Increment("reverse_proxy","429_busy_retry")
				goto START
			} else {
				log.Debugf("reached max retries, failed to proxy request: %v, %v", err, string(req.RequestURI()))
			}
		}else{
			log.Warnf("failed to proxy request: %v, %v, retried #%v", err, string(req.RequestURI()), retry)
		}

		//TODO if backend failure and after reached max retry, should save translog and mark the elasticsearch cluster to downtime, deny any new requests
		// the translog file should consider to contain dirty writes, could be used to do cross cluster check or manually operations recovery.

		myctx.SetContentType(util.ContentTypeJson)
		myctx.Response.SwapBody([]byte(fmt.Sprintf("{\"error\":true,\"message\":\"%v\"}",err.Error())))
		myctx.SetStatusCode(500)
	} else {
		if global.Env().IsDebug {
			log.Tracef("request [%v] [%v] [%v] [%v]", req.URI().String(), util.SubString(string(req.GetRawBody()), 0, 256), res.StatusCode(), util.SubString(string(res.GetRawBody()), 0, 256))
		}
	}

	if useClient{
		myctx.Response.SetStatusCode(res.StatusCode())
		myctx.Response.Header.SetContentTypeBytes(res.Header.ContentType())
		myctx.Response.SetBody(res.Body())

		compress,compressType:= res.IsCompressed()
		if compress{
			myctx.Response.Header.Set(fasthttp.HeaderContentEncoding,string(compressType))
		}

		fasthttp.ReleaseResponse(res)
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
