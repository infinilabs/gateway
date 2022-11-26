/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package entry

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/framework/lib/fasthttp/reuseport"
	r "infini.sh/framework/lib/router"
	"infini.sh/gateway/common"
	"net"
	"os"
	"path"
	"runtime"
	"strings"
	"time"
)

func NewEntrypoint(config common.EntryConfig) *Entrypoint {
	return &Entrypoint{
		config: config,
	}
}

type Entrypoint struct {
	config common.EntryConfig

	routerConfig common.RouterConfig

	certPool      *x509.CertPool
	rootCert      *x509.Certificate
	rootKey       *rsa.PrivateKey
	rootCertPEM   []byte
	schema string
	listenAddress string
	router        *r.Router
	server        *fasthttp.Server
}

func (this *Entrypoint) String() string {
	return fmt.Sprintf("%v", this.config.Name)
}

func (this *Entrypoint) Start() error {
	if !this.config.Enabled {
		return nil
	}

	if this.config.NetworkConfig.ReusePort == this.config.NetworkConfig.SkipOccupiedPort && this.config.NetworkConfig.ReusePort == true {
		return errors.New("port reuse and skip occupied can't be enabled at the same time for entry:" + this.config.Name)
	}

	this.listenAddress = this.config.NetworkConfig.GetBindingAddr()

	if !this.config.NetworkConfig.ReusePort && this.config.NetworkConfig.SkipOccupiedPort {
		this.listenAddress = util.AutoGetAddress(this.config.NetworkConfig.GetBindingAddr())
		log.Trace("auto skip address ", this.listenAddress)
	}

	var ln net.Listener
	var err error

	if this.config.NetworkConfig.ReusePort&&!strings.Contains(this.listenAddress,"::") {
		log.Debug("reuse port ", this.listenAddress)
		ln, err = reuseport.Listen("tcp4", this.config.NetworkConfig.GetBindingAddr())
	} else {
		ln, err = net.Listen("tcp", this.listenAddress)
	}

	if err != nil {
		panic(errors.Errorf("error in listener: %s", err))
	}

	this.router = r.New()

	if this.config.RouterConfigName != "" {
		this.routerConfig = common.GetRouter(this.config.RouterConfigName)
	}

	if len(this.routerConfig.Rules) > 0 {
		for _, rule := range this.routerConfig.Rules {

			flow := common.FilterFlow{}
			for _, y := range rule.Flow {

				cfg,err := common.GetFlowConfig(y)
				if err!=nil{
					panic(err)
				}

				if len(cfg.Filters) > 0 {
					flow1, err := pipeline.NewFilter(cfg.GetConfig())
					if err != nil {
						panic(err)
					}
					flow.JoinFilter(flow1)
				}
			}

			for _, v := range rule.Method {
				for _, u := range rule.PathPattern {
					log.Debugf("apply filter flow: [%s] [%s] [ %s ]", v, u, flow.ToString())
					if v == "*" {
						this.router.ANY(u, flow.Process)
					} else {
						this.router.Handle(v, u, flow.Process)
					}
				}
			}
		}
	}

	if this.routerConfig.DefaultFlow != "" {
		this.router.DefaultFlow = this.routerConfig.DefaultFlow
		if this.router.DefaultFlow!=""{
			//init func
			this.router.NotFound=common.GetFlowProcess(this.router.DefaultFlow)
		}
	} else {
		this.router.NotFound = func(ctx *fasthttp.RequestCtx) {
			ctx.Response.SetBody([]byte("NOT FOUND"))
			ctx.Response.SetStatusCode(404)
		}
	}

	if this.routerConfig.TracingFlow != "" {
		if global.Env().IsDebug {
			log.Debugf("tracing flow placed: %s", this.routerConfig.TracingFlow)
		}

		this.UpdateTracingFlow(this.routerConfig.TracingFlow)
	}

	if this.config.MaxConcurrency <= 0 {
		this.config.MaxConcurrency = 10000
	}

	if this.config.ReadTimeout <= 0 {
		this.config.ReadTimeout = 30
	}

	if this.config.IdleTimeout <= 0 {
		this.config.IdleTimeout = 30
	}

	if this.config.MaxIdleWorkerDurationSeconds <= 0 {
		this.config.MaxIdleWorkerDurationSeconds = 10
	}

	if this.config.TCPKeepaliveSeconds <= 0 {
		this.config.TCPKeepaliveSeconds = 15*60
	}

	if this.config.WriteTimeout <= 0 {
		this.config.WriteTimeout = 30
	}

	if this.config.SleepWhenConcurrencyLimitsExceeded <= 0 {
		this.config.SleepWhenConcurrencyLimitsExceeded = 10
	}

	if this.config.ReadBufferSize <= 0 {
		this.config.ReadBufferSize = 4 * 4096
	}

	if this.config.WriteBufferSize <= 0 {
		this.config.WriteBufferSize = 4 * 4096
	}

	if this.config.MaxRequestBodySize <= 0 {
		this.config.MaxRequestBodySize = 200 * 1024 * 1024
	}

	this.server = &fasthttp.Server{
		Name:                               "INFINI",
		NoDefaultServerHeader:              true,
		NoDefaultDate:                      true,
		NoDefaultContentType:               true,
		DisableHeaderNamesNormalizing:      true,
		DisablePreParseMultipartForm:       true,
		Handler:                            this.router.Handler,
		TraceHandler:                       this.router.TraceHandler,
		Concurrency:                        this.config.MaxConcurrency,
		LogAllErrors:                       false,
		MaxRequestBodySize:                 this.config.MaxRequestBodySize, //200 * 1024 * 1024,
		GetOnly:                            false,
		ReduceMemoryUsage:                  this.config.ReduceMemoryUsage,
		TCPKeepalive:                       this.config.TCPKeepalive,
		TCPKeepalivePeriod:                 time.Duration(this.config.TCPKeepaliveSeconds) * time.Second,
		MaxIdleWorkerDuration:              time.Duration(this.config.MaxIdleWorkerDurationSeconds) * time.Second,
		IdleTimeout:                        time.Duration(this.config.IdleTimeout) * time.Second,
		ReadTimeout:                        time.Duration(this.config.ReadTimeout) * time.Second,
		WriteTimeout:                       time.Duration(this.config.WriteTimeout) * time.Second,
		ReadBufferSize:                     this.config.ReadBufferSize, //16 * 1024,
		WriteBufferSize:                    this.config.WriteBufferSize,
		SleepWhenConcurrencyLimitsExceeded: time.Duration(this.config.SleepWhenConcurrencyLimitsExceeded) * time.Second,
		MaxConnsPerIP:                      this.config.MaxConnsPerIP,
	}

	if this.routerConfig.IPAccessRules.Enabled&&len(this.routerConfig.IPAccessRules.ClientIP.DeniedList) > 0 {
		log.Tracef("adding %v client ip to denied list", len(this.routerConfig.IPAccessRules.ClientIP.DeniedList))
		for _, ip := range this.routerConfig.IPAccessRules.ClientIP.DeniedList {
			this.server.AddBlackIPList(ip)
		}
	}

	if this.routerConfig.IPAccessRules.Enabled&&len(this.routerConfig.IPAccessRules.ClientIP.PermittedList) > 0 {
		log.Tracef("adding %v client ip to permitted list", len(this.routerConfig.IPAccessRules.ClientIP.PermittedList))
		for _, ip := range this.routerConfig.IPAccessRules.ClientIP.PermittedList {
			this.server.AddWhiteIPList(ip)
		}
	}

	if this.config.TLSConfig.TLSEnabled {
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
				//tls.TLS_AES_128_GCM_SHA256,
				//tls.TLS_AES_256_GCM_SHA384,
				//tls.TLS_RSA_WITH_AES_128_CBC_SHA,
				//tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
				//tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, // Go 1.8 only
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,   // Go 1.8 only
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		}

		var ca, cert, key string
		cert=this.config.TLSConfig.TLSCertFile
		key=this.config.TLSConfig.TLSKeyFile

		log.Trace("using tls connection")

		if cert != "" && key != "" {
			log.Debug("using pre-defined cert files")


		} else {
			ca = path.Join(global.Env().GetDataDir(), "certs", "root.cert")
			cert = path.Join(global.Env().GetDataDir(), "certs", "auto.cert")
			key = path.Join(global.Env().GetDataDir(), "certs", "auto.key")

			if !(util.FileExists(ca) && util.FileExists(cert) && util.FileExists(key)) {

				os.MkdirAll(path.Join(global.Env().GetDataDir(), "certs"), 0755)

				log.Info("auto generating cert files")
				this.rootCert, this.rootKey, this.rootCertPEM = util.GetRootCert()

				this.certPool = x509.NewCertPool()
				this.certPool.AppendCertsFromPEM(this.rootCertPEM)

				// create a key-pair for the server
				servKey, err := rsa.GenerateKey(rand.Reader, 2048)
				if err != nil {
					panic(err)
				}

				// create a template for the server
				servCertTmpl, err := util.GetCertTemplateWithSingleDomain(this.config.TLSConfig.DefaultDomain)
				if err != nil {
					panic(err)
				}

				servCertTmpl.KeyUsage = x509.KeyUsageDigitalSignature
				servCertTmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}

				// create a certificate which wraps the server's public key, sign it with the root private key
				_, servCertPEM, err := util.CreateCert(servCertTmpl, this.rootCert, &servKey.PublicKey, this.rootKey)
				if err != nil {
					panic(err)
				}

				// provide the private key and the cert
				servKeyPEM := pem.EncodeToMemory(&pem.Block{
					Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(servKey),
				})

				util.FilePutContentWithByte(ca, this.rootCertPEM)
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
			defer func() {
				if !global.Env().IsDebug {
					if r := recover(); r != nil {
						var v string
						switch r.(type) {
						case error:
							v = r.(error).Error()
						case runtime.Error:
							v = r.(runtime.Error).Error()
						case string:
							v = r.(string)
						}
						log.Error("error", v)
					}
				}
			}()

			if err := this.server.Serve(lnTls); err != nil {
				panic(errors.Errorf("error in server: %s", err))
			}
		}()

	} else {
		log.Trace("starting insecure server")
		go func() {
			defer func() {
				if !global.Env().IsDebug {
					if r := recover(); r != nil {
						var v string
						switch r.(type) {
						case error:
							v = r.(error).Error()
						case runtime.Error:
							v = r.(runtime.Error).Error()
						case string:
							v = r.(string)
						}
						log.Error("error", v)
					}
				}
			}()
			if err := this.server.Serve(ln); err != nil {
				panic(errors.Errorf("error in server: %s", err))
			}
		}()
	}

	err = util.WaitServerUp(this.listenAddress, 30*time.Second)
	if err != nil {
		panic(err)
	}

	log.Infof("entry [%s] listen at: %s%s", this.String(), this.GetSchema(), this.listenAddress)

	return nil
}

func (this *Entrypoint) GetSchema()string{
	if this.schema!=""{
		return this.schema
	}
	if this.config.TLSConfig.TLSEnabled{
		return "https://"
	}else{
		return "http://"
	}
}

func (this *Entrypoint) GetConfig() common.EntryConfig {
	return this.config
}

func (this *Entrypoint) Stop() error {
	log.Tracef("entry [%s] closed", this.String())
	if !this.config.Enabled {
		return nil
	}

	if this.config.DirtyShutdown {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Millisecond*5000))
		defer cancel()
		go func(ctx context.Context) {
			defer func() {
				if !global.Env().IsDebug {
					if r := recover(); r != nil {
						var v string
						switch r.(type) {
						case error:
							v = r.(error).Error()
						case runtime.Error:
							v = r.(runtime.Error).Error()
						case string:
							v = r.(string)
						}
						log.Error("error", v)
					}
				}
			}()
			this.server.Shutdown()
		}(ctx)

		select {
		case <-ctx.Done():
			log.Debug("entry shutdown successful")
		case <-time.After(time.Duration(time.Second * 120)):
			log.Debug("entry shutdown 5s timeout")
		}
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Millisecond*5000))
		defer cancel()

		go func(ctx context.Context) {
			defer func() {
				if !global.Env().IsDebug {
					if r := recover(); r != nil {
						var v string
						switch r.(type) {
						case error:
							v = r.(error).Error()
						case runtime.Error:
							v = r.(runtime.Error).Error()
						case string:
							v = r.(string)
						}
						log.Error("error", v)
					}
				}
			}()

			if r := recover(); r != nil {}
			ticker := time.NewTicker(3*time.Second)
			for {
				select {
				case <-ticker.C:
					time.Sleep(1*time.Second)
					if util.ContainStr(this.listenAddress,"0.0.0.0"){
						this.listenAddress=strings.Replace(this.listenAddress,"0.0.0.0","127.0.0.1",-1)
					}
					util.HttpGet(this.GetSchema()+this.listenAddress+"/favicon.ico")
				case <-ctx.Done():
					return
				}
			}
		}(ctx)

		if this.server!=nil{
			this.server.Shutdown()
		}
	}

	return nil
}

func (this *Entrypoint) Stats() util.MapStr {
	data := util.MapStr{
		"open_connections": this.server.GetOpenConnectionsCount(),
	}
	return data
}

func (this *Entrypoint) RefreshTracingFlow() {

	if this.router != nil {
		if this.router.TracingFlow != "" {
			this.router.TraceHandler = common.GetFlowProcess(this.routerConfig.TracingFlow)
			if this.server != nil {
				this.server.TraceHandler = this.router.TraceHandler
			}
		}
	}
}

func (this *Entrypoint) RefreshDefaultFlow() {

	if this.router != nil {
		if this.router.DefaultFlow != "" {
			this.router.NotFound = common.GetFlowProcess(this.routerConfig.DefaultFlow)
			if this.server != nil {
				this.server.Handler = this.router.NotFound
			}
		}
	}
}

func (this *Entrypoint) UpdateTracingFlow(flow string) {
	if flow != "" {
		if this.router != nil {
			this.router.TracingFlow = this.routerConfig.TracingFlow
			this.router.TraceHandler = common.GetFlowProcess(this.routerConfig.TracingFlow)
		}
		if this.server != nil {
			this.server.TraceHandler = this.router.TraceHandler
		}
	} else {
		if this.router != nil {
			this.router.TracingFlow = ""
			this.router.TraceHandler = nil
		}

		if this.server != nil {
			this.server.TraceHandler = nil
		}
	}

}
