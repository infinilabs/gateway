// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package ldap

import (
	"crypto/tls"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/shaj13/libcache"
	_ "github.com/shaj13/libcache/lru"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/framework/lib/guardian/auth"
	"infini.sh/framework/lib/guardian/auth/strategies/ldap"
	"infini.sh/gateway/common"
	"net/http"
	"time"
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
	MaxCacheItems   int     `config:"max_cache_items"`
	CacheTTL   string     `config:"cache_ttl"`

	BypassAPIKey bool `config:"bypass_api_key"`
	ldapQuery    auth.Strategy
}

func (filter *LDAPFilter) Name() string {
	return "ldap_auth"
}

func (filter *LDAPFilter) Filter(ctx *fasthttp.RequestCtx) {

	t := ctx.Request.ParseAuthorization()
	if t == "ApiKey" && filter.BypassAPIKey {
		log.Error("apiKEY")
		return
	}

	user, err := filter.ldapQuery.Authenticate(ctx.Context(), &ctx.Request)

	if err != nil {
		log.Debug("invalid credentials, ", err)
		ctx.Error(fasthttp.StatusMessage(fasthttp.StatusUnauthorized), fasthttp.StatusUnauthorized)
		ctx.Response.Header.Set("WWW-Authenticate", "Basic realm=Restricted")
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
	pipeline.RegisterFilterPluginWithConfigMetadata("ldap_auth",pipeline.FilterConfigChecked(NewLDAPFilter, pipeline.RequireFields("host","bind_dn","base_dn")),&LDAPFilter{})
}

func NewLDAPFilter(c *config.Config) (pipeline.Filter, error) {

	runner := LDAPFilter{
		Tls:            false,
		RequireGroup:   true,
		Port:           389,
		CacheTTL: "300s",
		UserFilter:     "(uid=%s)",
		GroupFilter:    "(memberUid=%s)",
		UidAttribute:   "uid",
		GroupAttribute: "cn",
	}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	cfg := ldap.Config{
		Host:           runner.Host,
		Port:           runner.Port,
		BindDN:         runner.BindDn,
		BindPassword:   runner.BindPassword,
		BaseDN:         runner.BaseDn,
		UserFilter:     runner.UserFilter,
		GroupFilter:    runner.GroupFilter,
		UIDAttribute:   runner.UidAttribute,
		GroupAttribute: runner.GroupAttribute,
	}

	if len(runner.Attributes) > 0 {
		cfg.Attributes = runner.Attributes
	}

	if runner.Tls {
		cfg.TLS = &tls.Config{InsecureSkipVerify: true}
	}

	cacheObj := libcache.LRU.New(runner.MaxCacheItems)
	if runner.CacheTTL!=""{
		cacheObj.SetTTL(util.GetDurationOrDefault(runner.CacheTTL,time.Minute*5))
	}
	cacheObj.RegisterOnExpired(func(key, _ interface{}) {
		cacheObj.Peek(key)
	})

	runner.ldapQuery = ldap.NewCached(&cfg,cacheObj)

	return &runner, nil
}
