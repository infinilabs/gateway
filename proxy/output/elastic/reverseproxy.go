package elastic

import (
	"crypto/tls"
	"fmt"
	"math/rand"
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
}

var hostClients = map[string]*fasthttp.HostClient{}
var clients = map[string]*fasthttp.Client{}

func isEndpointValid(node elastic.NodesInfo, cfg *ProxyConfig) bool {

	log.Tracef("valid endpoint %v", node.Http.PublishAddress)
	var hasExclude = false
	var hasInclude = false
	endpoint := node.Http.PublishAddress
	for _, v := range cfg.Filter.Hosts.Exclude {
		hasExclude = true
		if endpoint == v {
			log.Debugf("host [%v] in exclude list, mark as invalid", node.Http.PublishAddress)
			return false
		}
	}

	for _, v := range cfg.Filter.Hosts.Include {
		hasInclude = true
		if endpoint == v {
			log.Debugf("host [%v] in include list, mark as valid", node.Http.PublishAddress)
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
			log.Debugf("node [%v] role [%v] match exclude rule [%v], mark as invalid", node.Http.PublishAddress, node.Roles, v)
			return false
		}
	}

	for _, v := range cfg.Filter.Roles.Include {
		hasInclude = true
		if util.ContainsAnyInArray(v, node.Roles) {
			log.Debugf("node [%v] role [%v] match include rule [%v], mark as valid", node.Http.PublishAddress, node.Roles, v)
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
					log.Debugf("node [%v] tags [%v:%v] in exclude list, mark as invalid", node.Http.PublishAddress, k, v)
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
					log.Debugf("node [%v] tags [%v:%v] in include list, mark as valid", node.Http.PublishAddress, k, v)
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

	//metadata not changed
	if metadata != nil && (metadata.NodesTopologyVersion==p.lastNodesTopologyVersion&&metadata.NodesTopologyVersion>0){
		return
	}

	endpoints := []string{}
	checkMetadata := false
	if metadata != nil && len(metadata.Nodes) > 0 {

		oldV := p.lastNodesTopologyVersion
		p.lastNodesTopologyVersion = metadata.NodesTopologyVersion

		if oldV == p.lastNodesTopologyVersion {
			if global.Env().IsDebug {
				log.Trace("metadata.NodesTopologyVersion is equal")
			}
			return
		}

		checkMetadata = true
		for _, y := range metadata.Nodes {
			if !isEndpointValid(y, cfg) {
				continue
			}

			endpoints = append(endpoints, y.Http.PublishAddress)
		}
		log.Tracef("discovery %v nodes: [%v]", len(endpoints), util.JoinArray(endpoints, ", "))
	}

	if len(endpoints) == 0 {
		endpoints = append(endpoints, esConfig.GetHost())
		if checkMetadata {
			log.Warnf("no matched endpoint, fallback to seed: %v", endpoints)
		}
	}

	for _, endpoint := range endpoints {
		_, ok := hostClients[endpoint]
		if !ok {
			hostClients[endpoint] = &fasthttp.HostClient{
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
				IsTLS: esConfig.IsTLS(),
				TLSConfig: &tls.Config{
					InsecureSkipVerify: cfg.TLSInsecureSkipVerify,
				},
			}
		}

		_, ok = clients[endpoint]
		if !ok {
			clients[endpoint] = &fasthttp.Client{
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

	if len(hostClients) == 0 {
		log.Error("proxy upstream is empty")
		metadata.ReportFailure()
		return
	}

	//replace with new hostClients
	//TODO add locker
	p.bla = balancer.NewBalancer(ws)
	log.Infof("elasticsearch [%v] endpoints: [%v] => [%v]", esConfig.Name, util.JoinArray(p.endpoints, ", "), util.JoinArray(endpoints, ", "))
	p.endpoints = endpoints
	log.Trace(esConfig.Name, " elasticsearch client nodes refreshed")

}

func NewReverseProxy(cfg *ProxyConfig) *ReverseProxy {

	p := ReverseProxy{
		oldAddr:     "",
		proxyConfig: cfg,
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
	if hostClients == nil {
		panic("ReverseProxy has been closed")
	}

	if len(hostClients) == 0 ||len(p.endpoints)==0{
		log.Error("no upstream found")
		return false, nil, ""
	}

	if p.bla != nil {
		// bla has been opened
		idx := p.bla.Distribute()
		if idx >= len(p.endpoints) {
			log.Warn("invalid offset, ", idx, " vs ", len(hostClients), p.endpoints, ", random pick now")
			idx = 0
			goto RANDOM
		}

		// if len(p.bla.) != len(p.endpoints) {
		// 	log.Warn("hostClients != endpoints, ", len(hostClients), " vs ", len(p.endpoints), ", random pick now")
		// 	goto RANDOM
		// }

		e := p.endpoints[idx]
		c, ok := hostClients[e] //TODO, check client by endpoint
		if !ok {
			log.Error("client not found for: ", e)
		}

		return true, c, e
	}

RANDOM:
	//or go random way
	max := len(hostClients)
	seed := rand.Intn(max)
	if seed >= len(hostClients)||seed >= len(p.endpoints) {
		log.Warn("invalid upstream offset, reset to 0")
		seed = 0
	}
	e := p.endpoints[seed]
	c := hostClients[e]
	return true, c, e
}

func (p *ReverseProxy) getClient() (clientAvailable bool, client *fasthttp.Client, endpoint string) {

	if clients == nil {
		panic("ReverseProxy has been closed")
	}

	if len(clients) == 0 ||len(p.endpoints)==0{
		log.Error("no upstream found")
		return false, nil, ""
	}

	if p.bla != nil {
		// bla has been opened
		idx := p.bla.Distribute()
		if idx >= len(p.endpoints) {
			log.Warn("invalid offset, ", idx, " vs ", len(clients), p.endpoints, ", random pick now")
			idx = 0
			goto RANDOM
		}

		e := p.endpoints[idx]
		c, ok := clients[e] //TODO, check client by endpoint
		if !ok {
			log.Error("client not found for: ", e)
		}

		return true, c, e
	}

RANDOM:
	//or go random way
	max := len(clients)
	seed := rand.Intn(max)
	if seed >= len(clients)||seed >= len(p.endpoints) {
		log.Warn("invalid upstream offset, reset to 0")
		seed = 0
	}
	e := p.endpoints[seed]
	c := clients[e]
	return true, c, e
}

func cleanHopHeaders(req *fasthttp.Request) {
	for _, h := range hopHeaders {
		req.Header.Del(h)
	}
}

var failureMessage = []string{"connection refused", "connection reset", "no such host", "timed out", "Connection: close"}

func (p *ReverseProxy) DelegateRequest(elasticsearch string, cfg *elastic.ElasticsearchMetadata, myctx *fasthttp.RequestCtx) {

	stats.Increment("cache", "strike")

	retry := 0
START:



	req := &myctx.Request
	res := &myctx.Response

	cleanHopHeaders(req)

	var pc fasthttp.ClientAPI
	var ok bool
	var endpoint string
	//使用算法来获取合适的 client
	switch cfg.Config.ClientMode{
	case "client":
		ok, pc, endpoint = p.getClient()
		break
	case "host":
		ok, pc, endpoint = p.getHostClient()
		break
	//case "pipeline":
		//ok, pc, endpoint = p.getHostClient()
		//break
	default:
		ok, pc, endpoint = p.getClient()
	}

	if !ok {
		//TODO no client available, throw error directly
		log.Error("no client available")
		return
	}

	// modify schema，align with elasticsearch's schema
	orignalSchema:=string(req.URI().Scheme())
	useClient:=false
	if cfg.Config.GetSchema()!=orignalSchema{
		req.URI().SetScheme(cfg.Config.GetSchema())
		ok, pc, endpoint = p.getClient()
		res = fasthttp.AcquireResponse()
		useClient=true
	}

	if global.Env().IsDebug {
		log.Tracef("send request [%v] to upstream [%v]", req.URI().String(), endpoint)
	}

	if cfg.Config.TrafficControl != nil {
	RetryRateLimit:

		if cfg.Config.TrafficControl.MaxQpsPerNode > 0 {
			if !rate.GetRateLimiterPerSecond(cfg.Config.ID, endpoint+"max_qps", int(cfg.Config.TrafficControl.MaxQpsPerNode)).Allow() {
				if global.Env().IsDebug {
					log.Tracef("throttle request [%v] to upstream [%v]", req.URI().String(), myctx.RemoteAddr().String())
				}
				time.Sleep(10 * time.Millisecond)
				goto RetryRateLimit
			}
		}

		if cfg.Config.TrafficControl.MaxBytesPerNode > 0 {
			if !rate.GetRateLimiterPerSecond(cfg.Config.ID, endpoint+"max_bps", int(cfg.Config.TrafficControl.MaxBytesPerNode)).AllowN(time.Now(), req.GetRequestLength()) {
				if global.Env().IsDebug {
					log.Tracef("throttle request [%v] to upstream [%v]", req.URI().String(), myctx.RemoteAddr().String())
				}
				time.Sleep(10 * time.Millisecond)
				goto RetryRateLimit
			}
		}
	}


	req.URI().SetHost(endpoint)

	err := pc.Do(req, res)

	// restore schema
	req.URI().SetScheme(orignalSchema)

	if  err != nil {
		if util.ContainsAnyInArray(err.Error(), failureMessage) {
			//record translog, update failure ticket
			if global.Env().IsDebug {
				if !rate.GetRateLimiterPerSecond(cfg.Config.ID, endpoint+"on_error", 1).Allow() {
					log.Errorf("elasticsearch [%v][%v] is on fire now, %v", p.proxyConfig.Elasticsearch,endpoint,err)
					time.Sleep(1 * time.Second)
				}
			}
			cfg.ReportFailure()
			//server failure flow
		} else if res.StatusCode() == 429 {
			retry++
			if p.proxyConfig.maxRetryTimes > 0 && retry < p.proxyConfig.maxRetryTimes {
				if p.proxyConfig.retryDelayInMs > 0 {
					time.Sleep(time.Duration(p.proxyConfig.retryDelayInMs) * time.Millisecond)
				}
				goto START
			} else {
				log.Debugf("reached max retries, failed to proxy request: %v, %v", err, string(req.RequestURI()))
			}
		}else{
			log.Warnf("failed to proxy request: %v, %v, retried #%v", err, string(req.RequestURI()), retry)
		}

		//TODO if backend failure and after reached max retry, should save translog and mark the elasticsearch cluster to downtime, deny any new requests
		// the translog file should consider to contain dirty writes, could be used to do cross cluster check or manually operations recovery.
		myctx.Response.SetBody([]byte(err.Error()))
	} else {
		if global.Env().IsDebug {
			log.Tracef("request [%v] [%v] [%v]", req.URI().String(), res.StatusCode(), util.SubString(string(res.GetRawBody()), 0, 256))
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

	myctx.Response.Header.Set("CLUSTER", p.proxyConfig.Elasticsearch)

	if myctx.Has("elastic_cluster_name") {
		es1 := myctx.MustGetStringArray("elastic_cluster_name")
		myctx.Set("elastic_cluster_name", append(es1, elasticsearch))
	} else {
		myctx.Set("elastic_cluster_name", []string{elasticsearch})
	}

	myctx.Response.Header.Set("UPSTREAM", endpoint)

	myctx.SetDestination(endpoint)

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
