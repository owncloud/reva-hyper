package registry

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	rRegistry "github.com/cs3org/reva/v2/pkg/registry"
	consulr "github.com/go-micro/plugins/v4/registry/consul"
	etcdr "github.com/go-micro/plugins/v4/registry/etcd"
	kubernetesr "github.com/go-micro/plugins/v4/registry/kubernetes"
	mdnsr "github.com/go-micro/plugins/v4/registry/mdns"
	memr "github.com/go-micro/plugins/v4/registry/memory"
	natsr "github.com/go-micro/plugins/v4/registry/nats"
	"github.com/owncloud/ocis/v2/ocis-pkg/natsjsregistry"
	mRegistry "go-micro.dev/v4/registry"
	"go-micro.dev/v4/registry/cache"
)

const (
	_registryEnv        = "MICRO_REGISTRY"
	_registryAddressEnv = "MICRO_REGISTRY_ADDRESS"
)

var (
	_once sync.Once
	_reg  mRegistry.Registry
)

// Config is the config for a registry
type Config struct {
	Type         string        `mapstructure:"type"`
	Addresses    []string      `mapstructure:"addresses"`
	Username     string        `mapstructure:"username"`
	Password     string        `mapstructure:"password"`
	DisableCache bool          `mapstructure:"disable_cache"`
	RegisterTTL  time.Duration `mapstructure:"register_ttl"`
}

// Option allows configuring the registry
type Option func(*Config)

// Inmemory overrides env values to use an in-memory registry
func Inmemory() Option {
	return func(c *Config) {
		c.Type = "memory"
	}
}

// GetRegistry returns a configured micro registry based on Micro env vars.
// It defaults to mDNS, so mind that systems with mDNS disabled by default (i.e SUSE) will have a hard time
// and it needs to explicitly use etcd. Os awareness for providing a working registry out of the box should be done.
func GetRegistry(opts ...Option) mRegistry.Registry {
	_once.Do(func() {
		cfg := getEnvs(opts...)

		switch cfg.Type {
		default:
			fmt.Println("Attention: unknown registry type, using default nats-js-kv")
			fallthrough
		case "natsjs", "nats-js", "nats-js-kv": // for backwards compatibility - we will stick with one of those
			_reg = natsjsregistry.NewRegistry(
				mRegistry.Addrs(cfg.Addresses...),
				natsjsregistry.DefaultTTL(cfg.RegisterTTL),
			)
		case "memory":
			_reg = memr.NewRegistry()
			cfg.DisableCache = true // no cache needed for in-memory registry
		case "kubernetes":
			fmt.Println("Attention: kubernetes registry is deprecated, use nats-js-kv instead")
			_reg = kubernetesr.NewRegistry(
				mRegistry.Addrs(cfg.Addresses...),
			)
		case "nats":
			fmt.Println("Attention: nats registry is deprecated, use nats-js-kv instead")
			_reg = natsr.NewRegistry(
				mRegistry.Addrs(cfg.Addresses...),
				natsr.RegisterAction("put"),
			)
		case "etcd":
			fmt.Println("Attention: etcd registry is deprecated, use nats-js-kv instead")
			_reg = etcdr.NewRegistry(
				mRegistry.Addrs(cfg.Addresses...),
			)
		case "consul":
			fmt.Println("Attention: consul registry is deprecated, use nats-js-kv instead")
			_reg = consulr.NewRegistry(
				mRegistry.Addrs(cfg.Addresses...),
			)
		case "mdns":
			fmt.Println("Attention: mdns registry is deprecated, use nats-js-kv instead")
			_reg = mdnsr.NewRegistry()
		}

		// Disable cache if wanted
		if !cfg.DisableCache {
			_reg = cache.New(_reg, cache.WithTTL(30*time.Second))
		}

		// fixme: lazy initialization of reva registry, needs refactor to a explicit call per service
		_ = rRegistry.Init(_reg)
	})
	// always use cached registry to prevent registry
	// lookup for every request
	return _reg
}

func getEnvs(opts ...Option) *Config {
	cfg := &Config{
		Type:      "nats-js-kv",
		Addresses: []string{"127.0.0.1:9233"},
	}

	if s := os.Getenv(_registryEnv); s != "" {
		cfg.Type = s
	}

	if s := strings.Split(os.Getenv(_registryAddressEnv), ","); len(s) > 0 && s[0] != "" {
		cfg.Addresses = s
	}

	cfg.RegisterTTL = GetRegisterTTL()

	for _, o := range opts {
		o(cfg)
	}

	return cfg
}
