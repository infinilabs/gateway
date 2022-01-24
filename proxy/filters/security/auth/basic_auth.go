package auth

import (
	"encoding/base64"
	"fmt"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"strings"
)

// basicAuth returns the username and password provided in the request's
// Authorization header, if the request uses HTTP Basic Authentication.
// See RFC 2617, Section 2.
func basicAuth(ctx *fasthttp.RequestCtx) (username, password string, ok bool) {
	auth := ctx.Request.Header.PeekAny(fasthttp.AuthHeaderKeys)
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
func BasicAuth1(h fasthttp.RequestHandler, requiredUser, requiredPassword string) fasthttp.RequestHandler {
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

type BasicAuth struct {
	ValidUsers map[string]string `config:"valid_users"`
}

func (filter *BasicAuth) Name() string {
	return "basic_auth"
}

func (filter *BasicAuth) Filter(ctx *fasthttp.RequestCtx) {

	exists, user, pass := ctx.Request.ParseBasicAuth()

	if !exists {
		ctx.Error("Basic Authentication Required", 403)
		ctx.Finished()
		return
	}

	if len(filter.ValidUsers) > 0 {
		p, ok := filter.ValidUsers[util.UnsafeBytesToString(user)]
		if ok {
			if util.UnsafeBytesToString(pass) == p {
				return
			}
		}
	}

	ctx.Error("Basic Authentication Required", 403)
	ctx.Finished()

}

func init() {
	pipeline.RegisterFilterPlugin("basic_auth",NewBasicAuthFilter)
}

func NewBasicAuthFilter(c *config.Config) (pipeline.Filter, error) {

	runner := BasicAuth{}

	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the filter configuration : %s", err)
	}

	return &runner, nil
}
