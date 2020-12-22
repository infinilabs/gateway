package elastic

import (
	"crypto/tls"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/proxy/balancer"
	"math/rand"
	"net/http"
)

type ReverseProxy struct {
	oldAddr     string
	bla         balancer.IBalancer
	clients     []*fasthttp.HostClient
	proxyConfig *ProxyConfig
}

func NewReverseProxy(cfg *ProxyConfig) *ReverseProxy {

	p := ReverseProxy{
		oldAddr:     "",
		clients:     []*fasthttp.HostClient{},
		proxyConfig: cfg,
	}

	ws := []int{}

	esConfig := elastic.GetConfig(cfg.Elasticsearch)
	endpoints:=[]string{}

	if cfg.Discover.Enabled {
		nodes, err := elastic.GetClient(esConfig.Name).GetNodes()
		if err!=nil{
			panic(err)
		}

		for _,y:=range nodes.Nodes{
			endpoint:=y.(map[string]interface{})["http"].(map[string]interface{})["publish_address"].(string)
			endpoints=append(endpoints,endpoint)
		}
		log.Infof("discovery %v nodes: [%v]",len(nodes.Nodes),util.JoinArray(endpoints,", "))
	}else{
		endpoints=append(endpoints,esConfig.GetHost())
	}

	for _,endpoint:=range endpoints{

		client := &fasthttp.HostClient{
			Addr:                          endpoint,
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
			MaxConns:                      cfg.MaxConnection,
			MaxResponseBodySize:           cfg.MaxResponseBodySize,
			//TODO
			//MaxConnWaitTimeout: cfg.MaxConnWaitTimeout,
			//MaxConnDuration: cfg.MaxConnWaitTimeout,
			//MaxIdleConnDuration: cfg.MaxIdleConnDuration,
			//ReadTimeout: cfg.ReadTimeout,
			//WriteTimeout: cfg.ReadTimeout,
			//ReadBufferSize: cfg.ReadBufferSize,
			//WriteBufferSize: cfg.WriteBufferSize,
			//RetryIf: func(request *fasthttp.Request) bool {
			//
			//},
			IsTLS:                         esConfig.IsTLS(),
			TLSConfig: &tls.Config{
				InsecureSkipVerify: true, //TODO
			},
		}

		p.clients=append(p.clients,client)

		//get predefined weights
		w,o:=cfg.Weights[endpoint]
		if !o||w<=0{
			w=1
		}
		ws=append(ws,w)
	}

	if len(p.clients)==0{
		panic(errors.New("proxy upstream is empty"))
	}

	p.bla = balancer.NewBalancer(ws)
	return &p
}

func (p *ReverseProxy) getClient() *fasthttp.HostClient {
	if p.clients == nil {
		panic("ReverseProxy has been closed")
	}

	if p.bla != nil {
		// bla has been opened
		idx := p.bla.Distribute()
		c:= p.clients[idx]
		return c
	}

	//or go random way
	max:=len(p.clients)
	seed:=rand.Intn(max)
	c:= p.clients[seed]
	return c
}

func cleanHopHeaders(req *fasthttp.Request) {
	for _, h := range hopHeaders {
		req.Header.Del(h)
	}
}


func (p *ReverseProxy) DelegateRequest(req *fasthttp.Request, res *fasthttp.Response) {

	stats.Increment("cache", "strike")

	//使用算法来获取合适的 client
	pc := p.getClient()

	cleanHopHeaders(req)

	if global.Env().IsDebug{
		log.Tracef("send request [%v] to upstream [%v]",req.URI().String(),pc.Addr)
	}


	if err := pc.Do(req, res); err != nil {
		log.Errorf("failed to proxy request: %v, %v", err, string(req.RequestURI()))
		res.SetStatusCode(http.StatusInternalServerError)
		res.SetBodyRaw([]byte(err.Error()))
	}

	res.Header.Set("UPSTREAM",pc.Addr)

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
