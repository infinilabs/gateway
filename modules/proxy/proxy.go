package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	log "github.com/cihub/seelog"
	. "infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/framework/lib/fasthttp/reuseport"
	"infini.sh/gateway/config"
	"infini.sh/gateway/lib/translog"
	"infini.sh/gateway/modules/api"
	"infini.sh/gateway/modules/proxy/common"
	"infini.sh/gateway/modules/proxy/filter"
	proxy "infini.sh/gateway/modules/proxy/reverse-proxy"
	r "infini.sh/gateway/modules/proxy/router"
	"net"
	_ "net/http/pprof"
	"os"
	"path"
	"time"
)

func ProxyHandler(ctx *fasthttp.RequestCtx) {

	stats.Increment("request", "total")

	//# Traffic Control Layer
	//Phase: eBPF based IP filter

	//Phase: XDP based traffic control, forward 1%-100% to another node, can be used for warming up or a/b testing

	//# Route Layer
	//router:= router.NewRouter()
	//reqFlowID,resFlowID:=router.GetFlow(ctx)

	//Phase: Handle Parameters, remove customized parameters and setup context

	//# DAG based Request Processing Flow
	//if reqFlowID!=""{
	//	flow.GetFlow(reqFlowID).Process(ctx)
	//}
	//Phase: Requests Deny
	//TODO 根据请求IP和头信息,执行请求拒绝, 基于后台设置的黑白名单,执行准入, 只允许特定 IP Agent 访问 Gateway 访问

	//Phase: Deny Requests By Custom Rules, filter bad queries
	//TODO 慢查询,非法查询 主动检测和拒绝

	//Phase: Throttle Requests

	//Phase: Requests Decision
	//Phase: DAG based Process
	//自动学习请求网站来生成 FST 路由信息, 基于 FST 数来快速路由

	//# Delegate Requests to upstream
	proxyServer.DelegateRequest(&ctx.Request, &ctx.Response)

	//https://github.com/projectcontour/contour/blob/main/internal/dag/dag.go
	//Timeout Policy
	//Retry Policy
	//Virtual Policy
	//Routing Policy
	//Failback/Failsafe Policy

	//Phase: Handle Write Requests
	//Phase: Async Persist CUD

	//Phase: Cache Process
	//TODO, no_cache -> skip cache and del query_args

	//Phase: Request Rewrite, reset @timestamp precision for Kibana

	//# Response Processing Flow
	//if resFlowID!=""{
	//	flow.GetFlow(resFlowID).Process(ctx)
	//}
	//Phase: Recording
	//TODO 记录所有请求,采样记录,按条件记录

	//TODO 实时统计前后端 QPS, 出拓扑监控图
	//TODO 后台可以上传替换和编辑文件内容到缓存库里面, 直接返回自定义内容,如: favicon.ico, 可用于常用请求的提前预热,按 RequestURI 进行选择, 而不是完整 Hash

	//logging event
	//TODO configure log req and response, by condition
}

type ProxyModule struct {
}

func (this ProxyModule) Name() string {
	return "Proxy"
}

var (
	proxyConfig = config.ProxyConfig{
		MaxConcurrency:      1000,
		PassthroughPatterns: []string{"_cat", "scroll", "scroll_id", "_refresh", "_cluster", "_ccr", "_count", "_flush", "_ilm", "_ingest", "_license", "_migration", "_ml", "_nodes", "_rollup", "_data_stream", "_open", "_close"},
	}
)
var proxyServer *proxy.ReverseProxy

func (module ProxyModule) Setup(cfg *Config) {

	env.ParseConfig("proxy", &proxyConfig)

	if !proxyConfig.Enabled {
		return
	}

	config.SetProxyConfig(proxyConfig)

	api.Init()
	filter.Init()

	proxyServer = proxy.NewReverseProxy(&proxyConfig)

	//init router, and default handler to
	router = r.New()
	router.NotFound = proxyServer.DelegateToUpstream

	if global.Env().IsDebug{
		log.Trace("tracing enabled:", proxyConfig.TracingEnabled)
	}

	if proxyConfig.TracingEnabled {
		router.OnFinishHandler = common.GetFlowProcess("request_logging")
	}

}

func (module ProxyModule) Start() error {

	if !proxyConfig.Enabled {
		return nil
	}

	translog.Open()

	StartAPI()

	return nil
}

var certPool *x509.CertPool
var rootCert *x509.Certificate
var rootKey *rsa.PrivateKey
var rootCertPEM []byte
var listenAddress string
var router *r.Router

func StartAPI() {

	listenAddress = proxyConfig.NetworkConfig.GetBindingAddr()

	if !proxyConfig.NetworkConfig.ReusePort && proxyConfig.NetworkConfig.SkipOccupiedPort {
		listenAddress = util.AutoGetAddress(proxyConfig.NetworkConfig.GetBindingAddr())
	}

	var ln net.Listener
	var err error
	if proxyConfig.NetworkConfig.ReusePort {
		ln, err = reuseport.Listen("tcp4", proxyConfig.NetworkConfig.GetBindingAddr())
	} else {
		ln, err = net.Listen("tcp", listenAddress)
	}
	if err != nil {
		panic(errors.Errorf("error in listener: %s", err))
	}

	server := &fasthttp.Server{
		Name:                          "INFINI",
		DisableHeaderNamesNormalizing: true,
		Handler:                       router.Handler,
		Concurrency:                   proxyConfig.MaxConcurrency,
		LogAllErrors:                  false,
		MaxRequestBodySize:            200 * 1024 * 1024,
		GetOnly:                       false,
		ReduceMemoryUsage:             false,
		ReadTimeout:                   120 * time.Second,
		WriteTimeout:                  10 * time.Second,
		ReadBufferSize:                64 * 1024,
	}

	schema := "http://"
	if proxyConfig.TLSConfig.TLSEnabled {
		schema = "https://"
		cfg := &tls.Config{
			MinVersion: tls.VersionTLS12,
			CurvePreferences: []tls.CurveID{
				tls.CurveP256,
				tls.X25519, // Go 1.8 only
			},
			PreferServerCipherSuites: true,
			InsecureSkipVerify:       true,
			SessionTicketsDisabled:   false,
			ClientSessionCache:       tls.NewLRUClientSessionCache(128),
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, // Go 1.8 only
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,   // Go 1.8 only
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
			NextProtos: []string{"spdy/3"},
		}

		var ca, cert, key string
		log.Trace("using tls connection")

		if cert != "" && key != "" {
			log.Debug("using pre-defined cert files")

		} else {
			ca = path.Join(global.Env().GetWorkingDir(), "certs", "root.cert")
			cert = path.Join(global.Env().GetWorkingDir(), "certs", "auto.cert")
			key = path.Join(global.Env().GetWorkingDir(), "certs", "auto.key")

			if !(util.FileExists(ca) && util.FileExists(cert) && util.FileExists(key)) {

				os.MkdirAll(path.Join(global.Env().GetWorkingDir(), "certs"), 0755)

				log.Info("auto generating cert files")
				rootCert, rootKey, rootCertPEM = util.GetRootCert()

				certPool = x509.NewCertPool()
				certPool.AppendCertsFromPEM(rootCertPEM)

				// create a key-pair for the server
				servKey, err := rsa.GenerateKey(rand.Reader, 2048)
				if err != nil {
					panic(err)
				}

				// create a template for the server
				servCertTmpl, err := util.GetCertTemplate()
				if err != nil {
					panic(err)
				}

				servCertTmpl.KeyUsage = x509.KeyUsageDigitalSignature
				servCertTmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}

				// create a certificate which wraps the server's public key, sign it with the root private key
				_, servCertPEM, err := util.CreateCert(servCertTmpl, rootCert, &servKey.PublicKey, rootKey)
				if err != nil {
					panic(err)
				}

				// provide the private key and the cert
				servKeyPEM := pem.EncodeToMemory(&pem.Block{
					Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(servKey),
				})

				util.FilePutContentWithByte(ca, rootCertPEM)
				util.FilePutContentWithByte(cert, servCertPEM)
				util.FilePutContentWithByte(key, servKeyPEM)
			} else {
				log.Debug("loading auto generated certs")
			}
		}

		crt, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			panic(err)
		}

		cfg.Certificates = append(cfg.Certificates, crt)

		cfg.BuildNameToCertificate()

		lnTls := tls.NewListener(ln, cfg)

		go func() {
			if err := server.Serve(lnTls); err != nil {
				panic(errors.Errorf("error in fasthttp Server: %s", err))
			}
		}()

	} else {
		log.Trace("starting insecure proxy server")
		go func() {
			if err := server.Serve(ln); err != nil {
				panic(errors.Errorf("error in proxy Server: %s", err))
			}
		}()
	}

	err = util.WaitServerUp(listenAddress, 30*time.Second)
	if err != nil {
		panic(err)
	}

	log.Info("proxy server listen at: ", schema, listenAddress)
}

func (module ProxyModule) Stop() error {
	if !proxyConfig.Enabled {
		return nil
	}

	translog.Close()

	return nil
}
