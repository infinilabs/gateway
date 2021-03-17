package elastic

import (
	"crypto/tls"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/rate"
	"infini.sh/framework/core/stats"
	task2 "infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/proxy/balancer"
	"math/rand"
	"net/http"
	"time"
)

type ReverseProxy struct {
	oldAddr     string
	bla         balancer.IBalancer
	clients     []*fasthttp.HostClient
	proxyConfig *ProxyConfig
	endpoints   []string
	lastNodesTopologyVersion int
}

func isEndpointValid(node elastic.NodesInfo, cfg *ProxyConfig) bool {

	log.Tracef("valid endpoint %v",node.Http.PublishAddress)
	var hasExclude =false
	var hasInclude =false
	endpoint:=node.Http.PublishAddress
	for _, v := range cfg.Filter.Hosts.Exclude {
		hasExclude=true
		if endpoint==v{
			log.Debugf("host in exclude list, mark as invalid, %v",node.Http.PublishAddress)
			return false
		}
	}

	for _, v := range cfg.Filter.Hosts.Include {
		hasInclude=true
		if endpoint==v{
			log.Debugf("host in include list, mark as valid, %v",node.Http.PublishAddress)
			return true
		}
	}

	//no exclude and only have include, means white list mode
	if !hasExclude && hasInclude{
		return false
	}


	hasExclude=false
	hasInclude =false
	for _, v := range cfg.Filter.Roles.Exclude {
		hasExclude=true
		if util.ContainsAnyInArray(v,node.Roles){
			log.Debugf("node role %v match exclude rule [%v], mark as invalid, %v",node.Roles,v,node.Http.PublishAddress)
			return false
		}
	}

	for _, v := range cfg.Filter.Roles.Include {
		hasInclude=true
		if util.ContainsAnyInArray(v,node.Roles){
			log.Debugf("node role %v match include rule [%v], mark as valid, %v",node.Roles,v,node.Http.PublishAddress)
			return true
		}
	}

	if !hasExclude && hasInclude{
		return false
	}

	hasExclude=false
	hasInclude =false
	for _,o := range cfg.Filter.Tags.Exclude {
		hasExclude=true
		for k,v:=range o{
			v1,ok:=node.Attributes[k]
			if ok{
				if v1==v{
					log.Debugf("node tags [%v:%v] in exclude list, mark as invalid, %v",k,v,node.Http.PublishAddress)
					return false
				}
			}
		}
	}

	for _,o := range cfg.Filter.Tags.Include {
		hasInclude=true
		for k, v:=range o{
			v1,ok:=node.Attributes[k]
			if ok{
				if v1==v{
					log.Debugf("node tags [%v:%v] in include list, mark as valid, %v",k,v,node.Http.PublishAddress)
					return true
				}
			}
		}
	}

	if !hasExclude && hasInclude{
		return false
	}

	return true
}

func (p *ReverseProxy) refreshNodes(force bool) {

	if global.Env().IsDebug{
		log.Trace("elasticsearch client nodes refreshing")
	}
	cfg := p.proxyConfig
	metadata := elastic.GetMetadata(cfg.Elasticsearch)

	if metadata == nil && !force {
		log.Trace("metadata is nil and not forced, skip nodes refresh")
		return
	}

	ws := []int{}
	clients := []*fasthttp.HostClient{}
	esConfig := elastic.GetConfig(cfg.Elasticsearch)
	endpoints := []string{}

	checkMetadata:=false
	if metadata != nil && len(metadata.Nodes) > 0 {
		oldV:=p.lastNodesTopologyVersion
		p.lastNodesTopologyVersion=metadata.NodesTopologyVersion

		if oldV==p.lastNodesTopologyVersion {
			if global.Env().IsDebug{
				log.Trace("metadata.NodesTopologyVersion is equal")
			}
			return
		}

		checkMetadata=true
		for _, y := range metadata.Nodes {
			if !isEndpointValid(y, cfg) {
				continue
			}

			endpoints = append(endpoints, y.Http.PublishAddress)
		}
		log.Tracef("discovery %v nodes: [%v]", len(endpoints), util.JoinArray(endpoints, ", "))
	}
	if len(endpoints)==0{
		endpoints = append(endpoints, esConfig.GetHost())
		if checkMetadata{
			log.Warnf("no valid endpoint for elasticsearch, fallback to use the seed endpoint: [%v], please check filter rules",endpoints)
		}
	}

	for _, endpoint := range endpoints {
		client := &fasthttp.HostClient{
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
		clients = append(p.clients, client)
		//get predefined weights
		w, o := cfg.Weights[endpoint]
		if !o || w <= 0 {
			w = 1
		}
		ws = append(ws, w)
	}

	if len(clients) == 0 {
		log.Error("proxy upstream is empty")
		//panic(errors.New("proxy upstream is empty"))
		return
	}

	//replace with new clients
	p.clients = clients
	p.bla = balancer.NewBalancer(ws)
	log.Infof("elasticsearch [%v] endpoints: [%v] => [%v]", esConfig.Name, util.JoinArray(p.endpoints, ", "), util.JoinArray(endpoints, ", "))
	p.endpoints = endpoints
	log.Trace("elasticsearch client nodes refreshed")
}

func NewReverseProxy(cfg *ProxyConfig) *ReverseProxy {

	p := ReverseProxy{
		oldAddr:     "",
		clients:     []*fasthttp.HostClient{},
		proxyConfig: cfg,
	}

	p.refreshNodes(true)

	if cfg.Refresh.Enabled{
		log.Debugf("refresh enabled for elasticsearch: [%v]",cfg.Elasticsearch)
		task:=task2.ScheduleTask{
			Description:fmt.Sprintf("refresh nodes for elasticsearch [%v]",cfg.Elasticsearch),
			Type:"interval",
			Interval: cfg.Refresh.Interval,
			Task: func() {
				p.refreshNodes(false)
			},
		}
		task2.RegisterScheduleTask(task)
	}


	return &p
}

func (p *ReverseProxy) getClient() (client *fasthttp.HostClient,endpoint string) {
	if p.clients == nil {
		panic("ReverseProxy has been closed")
	}

	if p.bla != nil {
		// bla has been opened
		idx := p.bla.Distribute()
		if idx >= len(p.clients) {
			log.Tracef("invalid offset, reset to 0")
			idx = 0
		}
		c := p.clients[idx]
		e:=p.endpoints[idx]
		return c,e
	}

	//or go random way
	max := len(p.clients)
	seed := rand.Intn(max)
	if seed >= len(p.clients) {
		log.Warn("invalid upstream offset, reset to 0")
		seed = 0
	}
	c := p.clients[seed]
	e :=p.endpoints[seed]
	return c,e
}

func cleanHopHeaders(req *fasthttp.Request) {
	for _, h := range hopHeaders {
		req.Header.Del(h)
	}
}

var failureMessage=[]string{"connection refused","no such host","timed out"}

func (p *ReverseProxy) DelegateRequest(elasticsearch string,myctx *fasthttp.RequestCtx) {

	stats.Increment("cache", "strike")

	retry:=0
	START:

	//使用算法来获取合适的 client
	pc,endpoint := p.getClient()

	req:=&myctx.Request
	res:=&myctx.Response

	cleanHopHeaders(req)

	if global.Env().IsDebug {
		log.Tracef("send request [%v] to upstream [%v]", req.URI().String(), pc.Addr)
	}

	cfg:=elastic.GetConfig(elasticsearch)

	if cfg.TrafficControl!=nil{
	RetryRateLimit:

		if cfg.TrafficControl.MaxQpsPerNode>0{
			//fmt.Println("MaxQpsPerNode:",cfg.TrafficControl.MaxQpsPerNode)
			if !rate.GetRaterWithDefine(cfg.Name,endpoint+"max_qps", int(cfg.TrafficControl.MaxQpsPerNode)).Allow(){
				time.Sleep(10*time.Millisecond)
				goto RetryRateLimit
			}
		}

		if cfg.TrafficControl.MaxBytesPerNode>0{
			//fmt.Println("MaxBytesPerNode:",cfg.TrafficControl.MaxQpsPerNode)
			if !rate.GetRaterWithDefine(cfg.Name,endpoint+"max_bps", int(cfg.TrafficControl.MaxBytesPerNode)).AllowN(time.Now(),req.GetRequestLength()){
				time.Sleep(10*time.Millisecond)
				goto RetryRateLimit
			}
		}
	}

	if err := pc.Do(req, res); err != nil {
		log.Warnf("failed to proxy request: %v, %v, retried #%v", err, string(req.RequestURI()),retry)
		if util.ContainsAnyInArray(err.Error(),failureMessage){
			retry++
			if retry<10 {
				goto START
			}else{
				log.Debugf("reached max retries, failed to proxy request: %v, %v", err, string(req.RequestURI()))
			}
		}
		res.SetStatusCode(http.StatusInternalServerError)
		res.SetBody([]byte(err.Error()))
	}

	res.Header.Set("CLUSTER", p.proxyConfig.Elasticsearch)

	myctx.Set("elastic_cluster_name",elasticsearch)

	res.Header.Set("UPSTREAM", pc.Addr)
	res.SetDestination(pc.Addr)

}

// SetClient ...
func (p *ReverseProxy) SetClient(addr string) *ReverseProxy {
	for idx := range p.clients {
		p.clients[idx].Addr = addr
	}
	return p
}

// Reset ...
func (p *ReverseProxy) Reset() {
	for idx := range p.clients {
		p.clients[idx].Addr = ""
	}
}

// Close ... clear and release
func (p *ReverseProxy) Close() {
	p.clients = nil
	p.bla = nil
	p = nil
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
