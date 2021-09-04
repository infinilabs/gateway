/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package ldap

import (
	"crypto/tls"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/lib/guardian/auth/strategies/ldap"
	"net/http"
)

type LDAPFilter struct {
	param.Parameters
}

func (filter LDAPFilter) Name() string {
	return "ldap_auth"
}

func (filter LDAPFilter) Process(ctx *fasthttp.RequestCtx) {

	isTLS := filter.GetBool("tls", false)

	cfg := ldap.Config{
		Host:         filter.MustGetString("host"),
		Port:         filter.GetIntOrDefault("port", 389),
		BindDN:       filter.MustGetString("bind_dn"),
		BindPassword: filter.GetStringOrDefault("bind_password", ""),
		BaseDN:       filter.MustGetString("base_dn"),
		Filter:       filter.GetStringOrDefault("filter", ""),
	}

	if isTLS {
		cfg.TLS = &tls.Config{InsecureSkipVerify: true}
	}

	user, err := ldap.New(&cfg).Authenticate(ctx.Context(), &ctx.Request)

	if err != nil {
		log.Debug("invalid credentials, ", err)
		code := http.StatusUnauthorized
		ctx.SetStatusCode(code)
		ctx.SetBody([]byte(http.StatusText(code)))
		ctx.Finished()
		return
	}

	if global.Env().IsDebug {
		if user != nil {
			log.Trace(user)
		}
		log.Debugf("user %s success authenticated", user.GetUserName())
	}
}
