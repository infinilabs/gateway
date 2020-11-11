/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package entry

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/framework/lib/fasthttp/reuseport"
	"infini.sh/gateway/common"
	r "infini.sh/gateway/proxy/router"
	"net"
	"net/http"
	"os"
	"path"
	"time"
)

func NewEntrypoint(config common.EntryConfig) *Entrypoint{
	return &Entrypoint{
		config: config,
	}
}

type Entrypoint struct {

	config       common.EntryConfig

	routerConfig common.RouterConfig

	certPool      *x509.CertPool
	rootCert      *x509.Certificate
	rootKey       *rsa.PrivateKey
	rootCertPEM   []byte
	listenAddress string
	router        *r.Router
	server        *fasthttp.Server
}

func (this *Entrypoint) Name() string {
	return this.config.Name
}

func (this *Entrypoint) Start() error {
	if !this.config.Enabled {
		return nil
	}

	if this.config.NetworkConfig.ReusePort==this.config.NetworkConfig.SkipOccupiedPort&&this.config.NetworkConfig.ReusePort==true{
		return errors.New("port reuse and skip occupied can't be enabled at the same time for entry:"+this.config.Name)
	}

	this.listenAddress = this.config.NetworkConfig.GetBindingAddr()

	if !this.config.NetworkConfig.ReusePort && this.config.NetworkConfig.SkipOccupiedPort {
		this.listenAddress = util.AutoGetAddress(this.config.NetworkConfig.GetBindingAddr())
		log.Trace("auto skip address ",this.listenAddress)
	}

	var ln net.Listener
	var err error
	if this.config.NetworkConfig.ReusePort {
		log.Debug("reuse port ",this.listenAddress)
		ln, err = reuseport.Listen("tcp4", this.config.NetworkConfig.GetBindingAddr())
	} else {
		ln, err = net.Listen("tcp", this.listenAddress)
	}
	if err != nil {
		panic(errors.Errorf("error in listener: %s", err))
	}

	this.router=r.New()

	if this.config.RouterConfigName!=""{
		this.routerConfig=common.GetRouter(this.config.RouterConfigName)
	}

	if len(this.routerConfig.Rules)>0{
		for _,x:=range this.routerConfig.Rules{
			flow:=common.FilterFlow{}
			for _,y:=range x.Flow{
				cfg:=common.GetFlowConfig(y)
				for _,z:=range cfg.Filters{
					f:=common.GetFilterWithConfig(&z)
					flow.JoinFilter(f)
				}
			}
			for _,v:=range x.Method{
				for _,u:=range x.PathPattern{
					log.Debugf("apply rule: [%s] [%s] [ %s ]",v,u,flow.ToString())
					if v=="*"{
						this.router.Handle(http.MethodGet,u,flow.Process)
						this.router.Handle(http.MethodPost,u,flow.Process)
						this.router.Handle(http.MethodPut,u,flow.Process)
						this.router.Handle(http.MethodDelete,u,flow.Process)
						this.router.Handle(http.MethodHead,u,flow.Process)
					}else{
						this.router.Handle(v,u,flow.Process)
					}
				}
			}
		}
	}

	if this.routerConfig.NotFoundFlow!=""{
		this.router.NotFound=common.GetFlowProcess(this.routerConfig.NotFoundFlow)
	}else{
		this.router.NotFound= func(ctx *fasthttp.RequestCtx) {
			ctx.Response.SetBody([]byte("NOT FOUND"))
			ctx.Response.SetStatusCode(404)
		}
	}

	this.server = &fasthttp.Server{
		Name:                          "INFINI",
		DisableHeaderNamesNormalizing: true,
		Handler:                       this.router.Handler,
		Concurrency:                   this.config.MaxConcurrency,
		LogAllErrors:                  false,
		MaxRequestBodySize:            200 * 1024 * 1024,
		GetOnly:                       false,
		ReduceMemoryUsage:             false,
		ReadTimeout:                   30 * time.Second,
		WriteTimeout:                  30 * time.Second,
		ReadBufferSize:                64 * 1024,
	}

	schema := "http://"
	if this.config.TLSConfig.TLSEnabled {
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
				this.rootCert, this.rootKey, this.rootCertPEM = util.GetRootCert()

				this.certPool = x509.NewCertPool()
				this.certPool.AppendCertsFromPEM(this.rootCertPEM)

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
			if err := this.server.Serve(lnTls); err != nil {
				panic(errors.Errorf("error in server: %s", err))
			}
		}()

	} else {
		log.Trace("starting insecure server")
		go func() {
			if err := this.server.Serve(ln); err != nil {
				panic(errors.Errorf("error in server: %s", err))
			}
		}()
	}

	err = util.WaitServerUp(this.listenAddress, 30*time.Second)
	if err != nil {
		panic(err)
	}

	log.Infof("entry [%s] listen at: %s%s", this.Name(), schema, this.listenAddress)

	return nil
}

func (this *Entrypoint) Stop() error {
	log.Tracef("entry [%s] closed", this.Name())
	if !this.config.Enabled {
		return nil
	}
	return this.server.Shutdown()
	//translog.Close()

}
