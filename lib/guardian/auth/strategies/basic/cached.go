package basic

import (
	"context"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/gateway/lib/guardian/auth"
	"infini.sh/gateway/lib/guardian/auth/internal"
)

// ExtensionKey represents a key for the password in info extensions.
// Typically used when basic strategy cache the authentication decisions.
//
// Deprecated: No longer used.
const ExtensionKey = "x-go-guardian-basic-password"

// NewCached return new auth.Strategy.
// The returned strategy, caches the invocation result of authenticate function.
func NewCached(f AuthenticateFunc, cache auth.Cache, opts ...auth.Option) auth.Strategy {
	cb := new(cachedBasic)
	cb.fn = f
	cb.cache = cache
	cb.comparator = plainText{}
	cb.hasher = internal.PlainTextHasher{}
	for _, opt := range opts {
		opt.Apply(cb)
	}
	return New(cb.authenticate, opts...)
}

type entry struct {
	password string
	info     auth.Info
}

type cachedBasic struct {
	fn         AuthenticateFunc
	comparator Comparator
	cache      auth.Cache
	hasher     internal.Hasher
}

func (c *cachedBasic) authenticate(ctx context.Context, r *fasthttp.Request, userName, pass []byte) (auth.Info, error) { // nolint:lll
	hash := c.hasher.Hash(util.UnsafeBytesToString(userName))
	v, ok := c.cache.Load(hash)

	// if info not found invoke user authenticate function
	if !ok {
		return c.authenticatAndHash(ctx, r, hash, userName, pass)
	}

	ent, ok := v.(entry)
	if !ok {
		return nil, auth.NewTypeError("strategies/basic:", entry{}, v)
	}

	return ent.info, c.comparator.Compare(ent.password, util.UnsafeBytesToString(pass))
}

func (c *cachedBasic) authenticatAndHash(ctx context.Context, r *fasthttp.Request, hash string, userName, pass []byte) (auth.Info, error) { //nolint:lll
	info, err := c.fn(ctx, r, userName, pass)
	if err != nil {
		return nil, err
	}

	hashedPass, _ := c.comparator.Hash(util.UnsafeBytesToString(pass))
	ent := entry{
		password: hashedPass,
		info:     info,
	}
	c.cache.Store(hash, ent)

	return info, nil
}
