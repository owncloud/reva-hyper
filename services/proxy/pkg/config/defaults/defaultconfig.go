package defaults

import (
	"path"
	"strings"
	"time"

	"github.com/owncloud/ocis/v2/ocis-pkg/config/defaults"
	"github.com/owncloud/ocis/v2/ocis-pkg/shared"
	"github.com/owncloud/ocis/v2/ocis-pkg/structs"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/config"
)

// FullDefaultConfig returns a fully initialized default configuration
func FullDefaultConfig() *config.Config {
	cfg := DefaultConfig()
	EnsureDefaults(cfg)
	Sanitize(cfg)
	return cfg
}

// DefaultConfig returns a basic default configuration
func DefaultConfig() *config.Config {
	return &config.Config{
		Debug: config.Debug{
			Addr:  "127.0.0.1:9205",
			Token: "",
		},
		HTTP: config.HTTP{
			Addr:      "0.0.0.0:9200",
			Root:      "/",
			Namespace: "com.owncloud.web",
			TLSCert:   path.Join(defaults.BaseDataPath(), "proxy", "server.crt"),
			TLSKey:    path.Join(defaults.BaseDataPath(), "proxy", "server.key"),
			TLS:       true,
		},
		Service: config.Service{
			Name: "proxy",
		},
		OIDC: config.OIDC{
			Issuer: "https://localhost:9200",

			AccessTokenVerifyMethod: config.AccessTokenVerificationJWT,
			SkipUserInfo:            false,
			UserinfoCache: &config.Cache{
				Store:    "memory",
				Nodes:    []string{"127.0.0.1:9233"},
				Database: "cache-userinfo",
				TTL:      time.Second * 10,
			},
			JWKS: config.JWKS{
				RefreshInterval:   60, // minutes
				RefreshRateLimit:  60, // seconds
				RefreshTimeout:    10, // seconds
				RefreshUnknownKID: true,
			},
		},
		PolicySelector: nil,
		RoleAssignment: config.RoleAssignment{
			Driver: "default",
			// this default is only relevant when Driver is set to "oidc"
			OIDCRoleMapper: config.OIDCRoleMapper{
				RoleClaim: "roles",
				RolesMap: []config.RoleMapping{
					{RoleName: "admin", ClaimValue: "ocisAdmin"},
					{RoleName: "spaceadmin", ClaimValue: "ocisSpaceAdmin"},
					{RoleName: "user", ClaimValue: "ocisUser"},
					{RoleName: "user-light", ClaimValue: "ocisGuest"},
				},
			},
		},
		Reva: shared.DefaultRevaConfig(),
		PreSignedURL: config.PreSignedURL{
			AllowedHTTPMethods: []string{"GET"},
			Enabled:            true,
			SigningKeys: &config.SigningKeys{
				Store:              "nats-js-kv", // signing keys are written by ocs, so we cannot use memory. It is not shared.
				Nodes:              []string{"127.0.0.1:9233"},
				TTL:                time.Hour * 12,
				DisablePersistence: true,
			},
		},
		AccountBackend:        "cs3",
		UserOIDCClaim:         "preferred_username",
		UserCS3Claim:          "username",
		AutoprovisionAccounts: false,
		EnableBasicAuth:       false,
		InsecureBackends:      false,
	}
}

// DefaultPolicies returns the default proxy policies.
func DefaultPolicies() []config.Policy {
	const serviceWebCcs = "com.owncloud.web.ccs"
	return []config.Policy{
		{
			Name: "ocis",
			Routes: []config.Route{
				{
					Endpoint:    "/",
					Service:     "com.owncloud.web.web",
					Unprotected: true,
				},
				{
					Endpoint:    "/.well-known/webfinger",
					Service:     "com.owncloud.web.webfinger",
					Unprotected: true,
				},
				{
					Endpoint:    "/.well-known/openid-configuration",
					Service:     "com.owncloud.web.idp",
					Unprotected: true,
				},
				{
					Endpoint:    "/.well-known/caldav",
					Service:     serviceWebCcs,
					Unprotected: true,
				},
				{
					Endpoint:    "/.well-known/carddav",
					Service:     serviceWebCcs,
					Unprotected: true,
				},
				{
					Endpoint: "/branding/logo",
					Service:  "com.owncloud.web.web",
				},
				{
					Endpoint:    "/konnect/",
					Service:     "com.owncloud.web.idp",
					Unprotected: true,
				},
				{
					Endpoint:    "/signin/",
					Service:     "com.owncloud.web.idp",
					Unprotected: true,
				},
				{
					Endpoint: "/archiver",
					Service:  "com.owncloud.web.frontend",
				},
				{
					// reroute oc10 notifications endpoint to userlog service
					Endpoint: "/ocs/v2.php/apps/notifications/api/v1/notifications/sse",
					Service:  "com.owncloud.sse.sse",
				},
				{
					// reroute oc10 notifications endpoint to userlog service
					Endpoint: "/ocs/v2.php/apps/notifications/api/v1/notifications",
					Service:  "com.owncloud.userlog.userlog",
				},
				{
					Type:     config.RegexRoute,
					Endpoint: "/ocs/v[12].php/cloud/user/signing-key", // only `user/signing-key` is left in ocis-ocs
					Service:  "com.owncloud.web.ocs",
				},
				{
					Type:        config.RegexRoute,
					Endpoint:    "/ocs/v[12].php/config",
					Service:     "com.owncloud.web.frontend",
					Unprotected: true,
				},
				{
					Endpoint: "/sciencemesh/",
					Service:  "com.owncloud.web.ocm",
				},
				{
					Endpoint: "/ocm/",
					Service:  "com.owncloud.web.ocm",
				},
				{
					Endpoint: "/ocs/",
					Service:  "com.owncloud.web.frontend",
				},
				{
					Endpoint: "/ccs/",
					Service:  serviceWebCcs,
				},
				{
					Type:     config.QueryRoute,
					Endpoint: "/remote.php/?preview=1",
					Service:  "com.owncloud.web.webdav",
				},
				// TODO the actual REPORT goes to /dav/files/{username}, which is user specific ... how would this work in a spaces world?
				// TODO what paths are returned? the href contains the full path so it should be possible to return urls from other spaces?
				// TODO or we allow a REPORT on /dav/spaces to search all spaces and /dav/space/{spaceid} to search a specific space
				// send webdav REPORT requests to search service
				{
					Type:     config.RegexRoute,
					Method:   "REPORT",
					Endpoint: "(/remote.php)?/(web)?dav",
					Service:  "com.owncloud.web.webdav",
				},
				{
					Type:     config.QueryRoute,
					Endpoint: "/dav/?preview=1",
					Service:  "com.owncloud.web.webdav",
				},
				{
					Type:     config.QueryRoute,
					Endpoint: "/webdav/?preview=1",
					Service:  "com.owncloud.web.webdav",
				},
				{
					Endpoint: "/remote.php/",
					Service:  "com.owncloud.web.ocdav",
				},
				{
					Endpoint: "/dav/",
					Service:  "com.owncloud.web.ocdav",
				},
				{
					Endpoint: "/webdav/",
					Service:  "com.owncloud.web.ocdav",
				},
				{
					Endpoint:    "/status",
					Service:     "com.owncloud.web.ocdav",
					Unprotected: true,
				},
				{
					Endpoint:    "/status.php",
					Service:     "com.owncloud.web.ocdav",
					Unprotected: true,
				},
				{
					Endpoint: "/index.php/",
					Service:  "com.owncloud.web.ocdav",
				},
				{
					Endpoint: "/apps/",
					Service:  "com.owncloud.web.ocdav",
				},
				{
					Endpoint:    "/data",
					Service:     "com.owncloud.web.frontend",
					Unprotected: true,
				},
				{
					Endpoint:    "/app/list",
					Service:     "com.owncloud.web.frontend",
					Unprotected: true,
				},
				{
					Endpoint: "/app/", // /app or /apps? ocdav only handles /apps
					Service:  "com.owncloud.web.frontend",
				},
				{
					Endpoint: "/graph/v1.0/invitations",
					Service:  "com.owncloud.graph.invitations",
				},
				{
					Endpoint: "/graph/",
					Service:  "com.owncloud.graph.graph",
				},
				{
					Endpoint: "/api/v0/settings",
					Service:  "com.owncloud.web.settings",
				},
			},
		},
	}
}

// EnsureDefaults adds default values to the configuration if they are not set yet
func EnsureDefaults(cfg *config.Config) {
	// provide with defaults for shared logging, since we need a valid destination address for "envdecode".
	if cfg.Log == nil && cfg.Commons != nil && cfg.Commons.Log != nil {
		cfg.Log = &config.Log{
			Level:  cfg.Commons.Log.Level,
			Pretty: cfg.Commons.Log.Pretty,
			Color:  cfg.Commons.Log.Color,
			File:   cfg.Commons.Log.File,
		}
	} else if cfg.Log == nil {
		cfg.Log = &config.Log{}
	}
	// provide with defaults for shared tracing, since we need a valid destination address for "envdecode".
	if cfg.Tracing == nil && cfg.Commons != nil && cfg.Commons.Tracing != nil {
		cfg.Tracing = &config.Tracing{
			Enabled:   cfg.Commons.Tracing.Enabled,
			Type:      cfg.Commons.Tracing.Type,
			Endpoint:  cfg.Commons.Tracing.Endpoint,
			Collector: cfg.Commons.Tracing.Collector,
		}
	} else if cfg.Tracing == nil {
		cfg.Tracing = &config.Tracing{}
	}

	if cfg.OIDC.UserinfoCache == nil && cfg.Commons != nil && cfg.Commons.Cache != nil {
		cfg.OIDC.UserinfoCache = &config.Cache{
			Store: cfg.Commons.Cache.Store,
			Nodes: cfg.Commons.Cache.Nodes,
			Size:  cfg.Commons.Cache.Size,
		}
	} else if cfg.OIDC.UserinfoCache == nil {
		cfg.OIDC.UserinfoCache = &config.Cache{}
	}

	if cfg.MachineAuthAPIKey == "" && cfg.Commons != nil && cfg.Commons.MachineAuthAPIKey != "" {
		cfg.MachineAuthAPIKey = cfg.Commons.MachineAuthAPIKey
	}

	if cfg.Reva == nil && cfg.Commons != nil {
		cfg.Reva = structs.CopyOrZeroValue(cfg.Commons.Reva)
	}

	if cfg.GRPCClientTLS == nil && cfg.Commons != nil {
		cfg.GRPCClientTLS = structs.CopyOrZeroValue(cfg.Commons.GRPCClientTLS)
	}
}

// Sanitize sanitizes the configuration
func Sanitize(cfg *config.Config) {
	if cfg.Policies == nil {
		cfg.Policies = mergePolicies(DefaultPolicies(), cfg.AdditionalPolicies)
	}

	if cfg.PolicySelector == nil {
		cfg.PolicySelector = &config.PolicySelector{
			Static: &config.StaticSelectorConf{
				Policy: "ocis",
			},
		}
	}

	if cfg.HTTP.Root != "/" {
		cfg.HTTP.Root = strings.TrimSuffix(cfg.HTTP.Root, "/")
	}
}

func mergePolicies(policies []config.Policy, additionalPolicies []config.Policy) []config.Policy {
	for _, p := range additionalPolicies {
		found := false
		for i, po := range policies {
			if po.Name == p.Name {
				po.Routes = append(po.Routes, p.Routes...)
				policies[i] = po
				found = true
				break
			}
		}
		if !found {
			policies = append(policies, p)
		}
	}
	return policies
}
