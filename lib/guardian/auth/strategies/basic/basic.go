// Package basic provides authentication strategy,
// to authenticate HTTP requests using the standard basic scheme.
package basic

import (
	"context"
	"errors"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/lib/guardian/auth"
)

var (
	// ErrMissingPrams is returned by Authenticate Strategy method,
	// when failed to retrieve user credentials from request.
	ErrMissingPrams = errors.New("strategies/basic: Request missing BasicAuth")

	// ErrInvalidCredentials is returned by Authenticate Strategy method,
	// when user password is invalid.
	ErrInvalidCredentials = errors.New("strategies/basic: Invalid user credentials")
)

// AuthenticateFunc declare custom function to authenticate request using user credentials.
// the authenticate function invoked by Authenticate Strategy method after extracting user credentials
// to compare against DB or other service, if extracting user credentials from request failed a nil info
// with ErrMissingPrams returned, Otherwise, return Authenticate invocation result.
type AuthenticateFunc func(ctx context.Context, r *fasthttp.Request, userName, password []byte) (auth.Info, error)

type basic struct {
	fn     AuthenticateFunc
	parser Parser
}

func (b basic) Authenticate(ctx context.Context, r *fasthttp.Request) (auth.Info, error) {
	exists,user,pass:=r.ParseBasicAuth()
	if !exists{
		return nil,errors.New("failed to Authenticate")
	}
	return b.fn(ctx, r, user, pass)
}

// New return new auth.Strategy.
func New(fn AuthenticateFunc, opts ...auth.Option) auth.Strategy {
	b := new(basic)
	b.fn = fn
	//b.parser = AuthorizationParser()
	for _, opt := range opts {
		opt.Apply(b)
	}
	return b
}
