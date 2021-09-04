/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package ldap

import (
	"crypto/tls"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/util"
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
		UserFilter:       filter.GetStringOrDefault("user_filter", "(uid=%s)"),
		GroupFilter:       filter.GetStringOrDefault("group_filter", "(memberUid=%s)"),
		UIDAttribute:       filter.GetStringOrDefault("uid_attribute", "uid"),
		GroupAttribute:       filter.GetStringOrDefault("group_attribute", "cn"),
	}

	attrs,ok:= filter.GetStringArray("attributes")
	if ok{
		cfg.Attributes=attrs
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
		log.Debug("id:",user.GetID(),", username:",user.GetUserName(),", groups:",util.JoinArray(user.GetGroups()," => "))

		if user != nil {
			log.Trace(user)
		}
		log.Debugf("user %s success authenticated", user.GetUserName())
	}

	if filter.GetBool("require_group",true){
		if len(user.GetGroups())==0{
			log.Debug(user.GetUserName()," has no group")
			code := http.StatusUnauthorized
			ctx.SetStatusCode(code)
			ctx.SetBody([]byte("user has no group information"))
			ctx.Finished()
			return
		}
	}

}
