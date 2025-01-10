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

package ldap

import (
	"context"
	"fmt"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/lib/guardian/auth/strategies/ldap"
	"testing"
)

func TestLDAPFunctions(t *testing.T) {

	cfg := ldap.Config{
		BaseDN:       "dc=example,dc=com",
		BindDN:       "cn=read-only-admin,dc=example,dc=com",
		Port:         389,
		Host:         "ldap.forumsys.com",
		BindPassword: "password",
		UserFilter:   "(uid=%s)",
	}

	r := fasthttp.AcquireRequest()
	r.SetBasicAuth("galieleo", "password")

	user, err := ldap.New(&cfg).Authenticate(context.Background(), r)

	fmt.Println(err)
	fmt.Println(user.GetUserName())
	fmt.Println(user.GetID())
	fmt.Println(user.GetGroups())
	fmt.Println(user.GetExtensions())

}
