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

//go:build !ci

package ldap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"infini.sh/framework/lib/fasthttp"
	"infini.sh/framework/lib/guardian/auth/strategies/ldap"
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

	r := &fasthttp.Request{}
	r.SetBasicAuth("galieleo", "password")

	ldapClient := ldap.New(&cfg)
	user, err := ldapClient.Authenticate(context.Background(), r)

	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	assert.Equal(t, "galieleo", user.GetUserName(), "unexpected username")
	assert.Equal(t, "expected-id", user.GetID(), "unexpected user ID")
	assert.Equal(t, []string{"expected-group"}, user.GetGroups(), "unexpected user groups")
	assert.Equal(t, map[string]string{"key": "value"}, user.GetExtensions(), "unexpected user extensions")
}
