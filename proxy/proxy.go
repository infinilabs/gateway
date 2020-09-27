package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	log "github.com/cihub/seelog"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/reuseport"
	. "infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/stats"
	"infini.sh/framework/core/util"
	"infini.sh/gateway/api"
	"infini.sh/gateway/config"
	"infini.sh/gateway/pipelines"
	proxy "infini.sh/gateway/proxy/reverse-proxy"
	"infini.sh/gateway/translog"
	"net"
	_ "net/http/pprof"
	"os"
	"path"
	"strings"
	"time"
)

func ProxyHandler(ctx *fasthttp.RequestCtx) {

	stats.Increment("requests", "total")

	proxyServer.ServeHTTP(ctx)

	//logging event
	//TODO configure log req and response, by condition
	//fmt.Println(ctx.Response.StatusCode())
	//fmt.Println(string(ctx.Response.Body()))


}

type ProxyPlugin struct {
}

func (this ProxyPlugin) Name() string {
	return "Proxy"
}

var (
	proxyConfig = config.ProxyConfig{
		PassthroughPatterns: []string{
			"_search", "_count", "_analyze", "_mget",
			"_doc", "_mtermvectors", "_msearch", "_search_shards", "_suggest",
			"_validate", "_explain", "_field_caps", "_rank_eval", "_aliases",
			"_open", "_close"},
	}
)
var proxyServer *proxy.ReverseProxy

func (module ProxyPlugin) Setup(cfg *Config) {
	//cfg.Unpack(&proxyConfig)

	env.ParseConfig("proxy", &proxyConfig)

	if !proxyConfig.Enabled {
		return
	}

	config.SetProxyConfig(proxyConfig)

	api.InitAPI()

	//register pipeline joints
	pipeline.RegisterPipeJoint(pipelines.IndexJoint{})
	pipeline.RegisterPipeJoint(pipelines.LoggingJoint{})

	proxyServer = proxy.NewReverseProxy(&proxyConfig)

}

func (module ProxyPlugin) Start() error {

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

// basicAuth returns the username and password provided in the request's
// Authorization header, if the request uses HTTP Basic Authentication.
// See RFC 2617, Section 2.
func basicAuth(ctx *fasthttp.RequestCtx) (username, password string, ok bool) {
	auth := ctx.Request.Header.Peek("Authorization")
	if auth == nil {
		return
	}
	return parseBasicAuth(string(auth))
}

// parseBasicAuth parses an HTTP Basic Authentication string.
// "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==" returns ("Aladdin", "open sesame", true).
func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}

// BasicAuth is the basic auth handler
func BasicAuth(h fasthttp.RequestHandler, requiredUser, requiredPassword string) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		// Get the Basic Authentication credentials
		user, password, hasAuth := basicAuth(ctx)

		if hasAuth && user == requiredUser && password == requiredPassword {
			// Delegate request to the given handle
			h(ctx)
			return
		}
		// Request Basic Authentication otherwise
		ctx.Error(fasthttp.StatusMessage(fasthttp.StatusUnauthorized), fasthttp.StatusUnauthorized)
		ctx.Response.Header.Set("WWW-Authenticate", "Basic realm=Restricted")
	})
}

func StartAPI() {
	if proxyConfig.NetworkConfig.SkipOccupiedPort {
		listenAddress = util.AutoGetAddress(proxyConfig.NetworkConfig.GetBindingAddr())
	} else {
		listenAddress = proxyConfig.NetworkConfig.GetBindingAddr()
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
		Name: "INFINI",
		DisableHeaderNamesNormalizing:true,
		Handler:            ProxyHandler,
		Concurrency:        1000,
		LogAllErrors:       false,
		MaxRequestBodySize: 20 * 1024 * 1024,
		GetOnly:            false,
		ReduceMemoryUsage:  false,
		ReadTimeout:        120 * time.Second,
		WriteTimeout:       10 * time.Second,
		ReadBufferSize:     64 * 1024,
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

				os.MkdirAll(path.Join(global.Env().GetWorkingDir(), "certs"), 0775)

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
			//if err := fasthttp.Serve(lnTls, ProxyHandler); err != nil {
			//	panic(err)
			//}
			if err := server.Serve(lnTls); err != nil {
				panic(errors.Errorf("error in fasthttp Server: %s", err))
			}
		}()

	} else {
		log.Trace("starting insecure API server")
		go func() {
			//if err = fasthttp.Serve(ln, ProxyHandler); err != nil {
			//	panic(errors.Errorf("error in fasthttp Server: %s", err))
			//}

			if err := server.Serve(ln); err != nil {
				panic(errors.Errorf("error in fasthttp Server: %s", err))
			}
		}()
	}

	err = util.WaitServerUp(listenAddress, 30*time.Second)
	if err != nil {
		panic(err)
	}

	log.Info("proxy server listen at: ", schema, listenAddress)
}

func (module ProxyPlugin) Stop() error {
	if !proxyConfig.Enabled {
		return nil
	}

	translog.Close()

	return nil
}
