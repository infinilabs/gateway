/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package ldap

import (
	"crypto/tls"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/common"
	"infini.sh/gateway/lib/guardian/auth/strategies/ldap"
	"net/http"
)

type LDAPFilter struct {
	Tls            bool     `config:"tls"`
	Host           string   `config:"host"`
	Port           int      `config:"port"`
	BindDn         string   `config:"bind_dn"`
	BindPassword   string   `config:"bind_password"`
	BaseDn         string   `config:"base_dn"`
	UserFilter     string   `config:"user_filter"`
	GroupFilter    string   `config:"group_filter"`
	UidAttribute   string   `config:"uid_attribute"`
	GroupAttribute string   `config:"group_attribute"`
	Attributes     []string `config:"attributes"`
	RequireGroup   bool     `config:"require_group"`
}

func (filter *LDAPFilter) Name() string {
	return "ldap_auth"
}

func (filter *LDAPFilter) Filter(ctx *fasthttp.RequestCtx) {

	cfg := ldap.Config{
		Host:           filter.Host,
		Port:           filter.Port,
		BindDN:         filter.BindDn,
		BindPassword:   filter.BindPassword,
		BaseDN:         filter.BaseDn,
		UserFilter:     filter.UserFilter,
		GroupFilter:    filter.GroupFilter,
		UIDAttribute:   filter.UidAttribute,
		GroupAttribute: filter.GroupAttribute,
	}

	if len(filter.Attributes) > 0 {
		cfg.Attributes = filter.Attributes
	}

	if filter.Tls {
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
		log.Debug("id:", user.GetID(), ", username:", user.GetUserName(), ", groups:", util.JoinArray(user.GetGroups(), " => "))

		if user != nil {
			log.Trace(user)
		}
		log.Debugf("user %s success authenticated", user.GetUserName())
	}

	if filter.RequireGroup {
		if len(user.GetGroups()) == 0 {
			log.Debug(user.GetUserName(), " has no group")
			code := http.StatusUnauthorized
			ctx.SetStatusCode(code)
			ctx.SetBody([]byte("user has no group information"))
			ctx.Finished()
			return
		}
	}

	ctx.Set("user_id", user.GetID())
	ctx.Set("user_name", user.GetUserName())
	ctx.Set("user_roles", common.GetLDAPGroupsMappingRoles(user.GetGroups()))

}

func init() {
	pipeline.RegisterFilterPlugin("ldap_auth",pipeline.FilterConfigChecked(NewLDAPFilter, pipeline.RequireFields("host","bind_dn","base_dn")))
}

func NewLDAPFilter(c *config.Config) (pipeline.Filter, error) {

	runner := LDAPFilter{
		Tls:            false,
		RequireGroup:   true,
		Port:           389,
		UserFilter:     "(uid=%s)",
		GroupFilter:    "(memberUid=%s)",
		UidAttribute:   "uid",
		GroupAttribute: "cn",
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
