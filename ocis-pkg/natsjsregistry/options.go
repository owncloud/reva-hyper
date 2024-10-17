package natsjsregistry

import (
	"context"
	"time"

	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/store"
)

type storeOptionsKey struct{}
type defaultTTLKey struct{}

// StoreOptions sets the options for the underlying store
func StoreOptions(opts []store.Option) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, storeOptionsKey{}, opts)
	}
}

// DefaultTTL allows setting a default register TTL for services
func DefaultTTL(t time.Duration) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, defaultTTLKey{}, t)
	}
}
