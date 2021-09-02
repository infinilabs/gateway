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
		Port:         "389",
		Host:         "ldap.forumsys.com",
		BindPassword: "password",
		Filter:       "(uid=%s)",
	}

	r:=fasthttp.AcquireRequest()
	r.SetBasicAuth("galieleo", "password")

	user, err := ldap.New(&cfg).Authenticate(context.Background(), r)

	fmt.Println(err)
	fmt.Println(user.GetUserName())
	fmt.Println(user.GetID())
	fmt.Println(user.GetGroups())
	fmt.Println(user.GetExtensions())


}