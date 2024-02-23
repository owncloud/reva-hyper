package config

import (
	"context"
	"time"

	"github.com/owncloud/ocis/v2/ocis-pkg/shared"
	"go-micro.dev/v4/client"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `mask:"struct" yaml:"-"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`

	Tracing *Tracing `yaml:"tracing"`
	Log     *Log     `yaml:"log"`
	Debug   Debug    `mask:"struct" yaml:"debug"`

	HTTP HTTP `yaml:"http"`

	Reva          *shared.Reva          `yaml:"reva"`
	GRPCClientTLS *shared.GRPCClientTLS `yaml:"grpc_client_tls"`
	GrpcClient    client.Client         `yaml:"-"`

	RoleQuotas            map[string]uint64  `yaml:"role_quotas"`
	Policies              []Policy           `yaml:"policies"`
	AdditionalPolicies    []Policy           `yaml:"additional_policies"`
	OIDC                  OIDC               `yaml:"oidc"`
	ServiceAccount        ServiceAccount     `yaml:"service_account"`
	RoleAssignment        RoleAssignment     `yaml:"role_assignment"`
	PolicySelector        *PolicySelector    `yaml:"policy_selector"`
	PreSignedURL          PreSignedURL       `yaml:"pre_signed_url"`
	ClientAPIKeyStore     ClientAPIKeyStore  `yaml:"client_api_key_store"`
	AccountBackend        string             `yaml:"account_backend" env:"PROXY_ACCOUNT_BACKEND_TYPE" desc:"Account backend the PROXY service should use. Currently only 'cs3' is possible here."`
	UserOIDCClaim         string             `yaml:"user_oidc_claim" env:"PROXY_USER_OIDC_CLAIM" desc:"The name of an OpenID Connect claim that is used for resolving users with the account backend. The value of the claim must hold a per user unique, stable and non re-assignable identifier. The availability of claims depends on your Identity Provider. There are common claims available for most Identity providers like 'email' or 'preferred_username' but you can also add your own claim."`
	UserCS3Claim          string             `yaml:"user_cs3_claim" env:"PROXY_USER_CS3_CLAIM" desc:"The name of a CS3 user attribute (claim) that should be mapped to the 'user_oidc_claim'. Supported values are 'username', 'mail' and 'userid'."`
	MachineAuthAPIKey     string             `mask:"password" yaml:"machine_auth_api_key" env:"OCIS_MACHINE_AUTH_API_KEY;PROXY_MACHINE_AUTH_API_KEY" desc:"Machine auth API key used to validate internal requests necessary to access resources from other services."`
	AutoprovisionAccounts bool               `yaml:"auto_provision_accounts" env:"PROXY_AUTOPROVISION_ACCOUNTS" desc:"Set this to 'true' to automatically provision users that do not yet exist in the users service on-demand upon first sign-in. To use this a write-enabled libregraph user backend needs to be setup an running."`
	EnableBasicAuth       bool               `yaml:"enable_basic_auth" env:"PROXY_ENABLE_BASIC_AUTH" desc:"Set this to true to enable 'basic authentication' (username/password)."`
	InsecureBackends      bool               `yaml:"insecure_backends" env:"PROXY_INSECURE_BACKENDS" desc:"Disable TLS certificate validation for all HTTP backend connections."`
	BackendHTTPSCACert    string             `yaml:"backend_https_cacert" env:"PROXY_HTTPS_CACERT" desc:"Path/File for the root CA certificate used to validate the server’s TLS certificate for https enabled backend services."`
	AuthMiddleware        AuthMiddleware     `yaml:"auth_middleware"`
	PoliciesMiddleware    PoliciesMiddleware `yaml:"policies_middleware"`

	Context context.Context `yaml:"-" json:"-"`
}

// Policy enables us to use multiple directors.
type Policy struct {
	Name   string  `yaml:"name"`
	Routes []Route `yaml:"routes"`
}

// Route defines forwarding routes
type Route struct {
	Type RouteType `yaml:"type,omitempty"`
	// Method optionally limits the route to this HTTP method
	Method   string `yaml:"method,omitempty"`
	Endpoint string `yaml:"endpoint,omitempty"`
	// Backend is a static URL to forward the request to
	Backend string `yaml:"backend,omitempty"`
	// Service name to look up in the registry
	Service     string `yaml:"service,omitempty"`
	ApacheVHost bool   `yaml:"apache_vhost,omitempty"`
	Unprotected bool   `yaml:"unprotected,omitempty"`
}

// RouteType defines the type of a route
type RouteType string

const (
	// PrefixRoute are routes matched by a prefix
	PrefixRoute RouteType = "prefix"
	// QueryRoute are routes matched by a prefix and query parameters
	QueryRoute RouteType = "query"
	// RegexRoute are routes matched by a pattern
	RegexRoute RouteType = "regex"
	// DefaultRouteType is the PrefixRoute
	DefaultRouteType RouteType = PrefixRoute
)

var (
	// RouteTypes is an array of the available route types
	RouteTypes = []RouteType{QueryRoute, RegexRoute, PrefixRoute}
)

// AuthMiddleware configures the proxy http auth middleware.
type AuthMiddleware struct {
	CredentialsByUserAgent map[string]string `yaml:"credentials_by_user_agent"`
}

// PoliciesMiddleware configures the proxy policies middleware.
type PoliciesMiddleware struct {
	Query string `yaml:"query" env:"PROXY_POLICIES_QUERY" desc:"Defines the 'Complete Rules' variable defined in the rego rule set this step uses for its evaluation. Rules default to deny if the variable was not found."`
}

const (
	AccessTokenVerificationNone = "none"
	AccessTokenVerificationJWT  = "jwt"
	// tdb:
	// AccessTokenVerificationIntrospect = "introspect"
)

// OIDC is the config for the OpenID-Connect middleware. If set the proxy will try to authenticate every request
// with the configured oidc-provider
type OIDC struct {
	Issuer                  string `yaml:"issuer" env:"OCIS_URL;OCIS_OIDC_ISSUER;PROXY_OIDC_ISSUER" desc:"URL of the OIDC issuer. It defaults to URL of the builtin IDP."`
	Insecure                bool   `yaml:"insecure" env:"OCIS_INSECURE;PROXY_OIDC_INSECURE" desc:"Disable TLS certificate validation for connections to the IDP. Note that this is not recommended for production environments."`
	AccessTokenVerifyMethod string `yaml:"access_token_verify_method" env:"PROXY_OIDC_ACCESS_TOKEN_VERIFY_METHOD" desc:"Sets how OIDC access tokens should be verified. Possible values are 'none' and 'jwt'. When using 'none', no special validation apart from using it for accessing the IPD's userinfo endpoint will be done. When using 'jwt', it tries to parse the access token as a jwt token and verifies the signature using the keys published on the IDP's 'jwks_uri'."`
	SkipUserInfo            bool   `yaml:"skip_user_info" env:"PROXY_OIDC_SKIP_USER_INFO" desc:"Do not look up user claims at the userinfo endpoint and directly read them from the access token. Incompatible with 'PROXY_OIDC_ACCESS_TOKEN_VERIFY_METHOD=none'."`
	UserinfoCache           *Cache `yaml:"user_info_cache"`
	JWKS                    JWKS   `yaml:"jwks"`
	RewriteWellKnown        bool   `yaml:"rewrite_well_known" env:"PROXY_OIDC_REWRITE_WELLKNOWN" desc:"Enables rewriting the /.well-known/openid-configuration to the configured OIDC issuer. Needed by the Desktop Client, Android Client and iOS Client to discover the OIDC provider."`
}

type JWKS struct {
	RefreshInterval   uint64 `yaml:"refresh_interval" env:"PROXY_OIDC_JWKS_REFRESH_INTERVAL" desc:"The interval for refreshing the JWKS (JSON Web Key Set) in minutes in the background via a new HTTP request to the IDP."`
	RefreshTimeout    uint64 `yaml:"refresh_timeout" env:"PROXY_OIDC_JWKS_REFRESH_TIMEOUT" desc:"The timeout in seconds for an outgoing JWKS request."`
	RefreshRateLimit  uint64 `yaml:"refresh_limit" env:"PROXY_OIDC_JWKS_REFRESH_RATE_LIMIT" desc:"Limits the rate in seconds at which refresh requests are performed for unknown keys. This is used to prevent malicious clients from imposing high network load on the IDP via ocis."`
	RefreshUnknownKID bool   `yaml:"refresh_unknown_kid" env:"PROXY_OIDC_JWKS_REFRESH_UNKNOWN_KID" desc:"If set to 'true', the JWKS refresh request will occur every time an unknown KEY ID (KID) is seen. Always set a 'refresh_limit' when enabling this."`
}

// Cache is a TTL cache configuration.
type Cache struct {
	Store              string        `yaml:"store" env:"OCIS_CACHE_STORE;PROXY_OIDC_USERINFO_CACHE_STORE" desc:"The type of the cache store. Supported values are: 'memory', 'redis-sentinel', 'nats-js-kv', 'noop'. See the text description for details."`
	Nodes              []string      `yaml:"addresses" env:"OCIS_CACHE_STORE_NODES;PROXY_OIDC_USERINFO_CACHE_STORE_NODES" desc:"A list of nodes to access the configured store. This has no effect when 'memory' or 'ocmem' stores are configured. Note that the behaviour how nodes are used is dependent on the library of the configured store. See the Environment Variable Types description for more details."`
	Database           string        `yaml:"database" env:"OCIS_CACHE_DATABASE" desc:"The database name the configured store should use."`
	Table              string        `yaml:"table" env:"PROXY_OIDC_USERINFO_CACHE_TABLE" desc:"The database table the store should use."`
	TTL                time.Duration `yaml:"ttl" env:"OCIS_CACHE_TTL;PROXY_OIDC_USERINFO_CACHE_TTL" desc:"Default time to live for user info in the user info cache. Only applied when access tokens has no expiration. See the Environment Variable Types description for more details."`
	Size               int           `yaml:"size" env:"OCIS_CACHE_SIZE;PROXY_OIDC_USERINFO_CACHE_SIZE" desc:"The maximum quantity of items in the user info cache. Only applies when store type 'ocmem' is configured. Defaults to 512 which is derived from the ocmem package though not exclicitely set as default."`
	DisablePersistence bool          `yaml:"disable_persistence" env:"OCIS_CACHE_DISABLE_PERSISTENCE;PROXY_OIDC_USERINFO_CACHE_DISABLE_PERSISTENCE" desc:"Disables persistence of the cache. Only applies when store type 'nats-js-kv' is configured. Defaults to false."`
	AuthUsername       string        `yaml:"username" env:"OCIS_CACHE_AUTH_USERNAME;PROXY_OIDC_USERINFO_CACHE_AUTH_USERNAME" desc:"The username to authenticate with the cache. Only applies when store type 'nats-js-kv' is configured."`
	AuthPassword       string        `yaml:"password" env:"OCIS_CACHE_AUTH_PASSWORD;PROXY_OIDC_USERINFO_CACHE_AUTH_PASSWORD" desc:"The password to authenticate with the cache. Only applies when store type 'nats-js-kv' is configured."`
}

// ClientAPIKeyStore is a TTL cache configuration.
type ClientAPIKeyStore struct {
	Store              string        `yaml:"store" env:"OCIS_CACHE_STORE;PROXY_CLIENT_API_KEY_STORE" desc:"The type of the cache store. Supported values are: 'memory', 'redis-sentinel', 'nats-js-kv', 'noop'. See the text description for details."`
	Nodes              []string      `yaml:"addresses" env:"OCIS_CACHE_STORE_NODES;PROXY_CLIENT_API_KEY_STORE_NODES" desc:"A list of nodes to access the configured store. This has no effect when 'memory' or 'ocmem' stores are configured. Note that the behaviour how nodes are used is dependent on the library of the configured store. See the Environment Variable Types description for more details."`
	TTL                time.Duration `yaml:"ttl" env:"OCIS_CACHE_TTL;PROXY_CLIENT_API_KEY_TTL" desc:"Default time to live for user info in the user info cache. Only applied when access tokens has no expiration. See the Environment Variable Types description for more details."`
	Size               int           `yaml:"size" env:"OCIS_CACHE_SIZE;PROXY_CLIENT_API_KEY_SIZE" desc:"The maximum quantity of items in the user info cache. Only applies when store type 'ocmem' is configured. Defaults to 512 which is derived from the ocmem package though not exclicitely set as default."`
	DisablePersistence bool          `yaml:"disable_persistence" env:"OCIS_CACHE_DISABLE_PERSISTENCE;PROXY_CLIENT_API_KEY_DISABLE_PERSISTENCE" desc:"Disables persistence of the cache. Only applies when store type 'nats-js-kv' is configured. Defaults to false."`
	AuthUsername       string        `yaml:"username" env:"OCIS_CACHE_AUTH_USERNAME;PROXY_CLIENT_API_KEY_AUTH_USERNAME" desc:"The username to authenticate with the cache. Only applies when store type 'nats-js-kv' is configured."`
	AuthPassword       string        `yaml:"password" env:"OCIS_CACHE_AUTH_PASSWORD;PROXY_CLIENT_API_KEY_AUTH_PASSWORD" desc:"The password to authenticate with the cache. Only applies when store type 'nats-js-kv' is configured."`
}

// RoleAssignment contains the configuration for how to assign roles to users during login
type RoleAssignment struct {
	Driver         string         `yaml:"driver" env:"PROXY_ROLE_ASSIGNMENT_DRIVER" desc:"The mechanism that should be used to assign roles to user upon login. Supported values: 'default' or 'oidc'. 'default' will assign the role 'user' to users which don't have a role assigned at the time they login. 'oidc' will assign the role based on the value of a claim (configured via PROXY_ROLE_ASSIGNMENT_OIDC_CLAIM) from the users OIDC claims."`
	OIDCRoleMapper OIDCRoleMapper `yaml:"oidc_role_mapper"`
}

// OIDCRoleMapper contains the configuration for the "oidc" role assignment driber
type OIDCRoleMapper struct {
	RoleClaim string        `yaml:"role_claim" env:"PROXY_ROLE_ASSIGNMENT_OIDC_CLAIM" desc:"The OIDC claim used to create the users role assignment."`
	RolesMap  []RoleMapping `yaml:"role_mapping" desc:"A list of mappings of ocis role names to PROXY_ROLE_ASSIGNMENT_OIDC_CLAIM claim values. This setting can only be configured in the configuration file and not via environment variables."`
}

// RoleMapping defines which ocis role matches a specific claim value
type RoleMapping struct {
	RoleName   string `yaml:"role_name" desc:"The name of an ocis role that this mapping should apply for."`
	ClaimValue string `yaml:"claim_value" desc:"The value of the 'PROXY_ROLE_ASSIGNMENT_OIDC_CLAIM' that matches the role defined in 'role_name'."`
}

// PolicySelector is the toplevel-configuration for different selectors
type PolicySelector struct {
	Static *StaticSelectorConf `yaml:"static"`
	Claims *ClaimsSelectorConf `yaml:"claims"`
	Regex  *RegexSelectorConf  `yaml:"regex"`
}

// StaticSelectorConf is the config for the static-policy-selector
type StaticSelectorConf struct {
	Policy string `yaml:"policy"`
}

// PreSignedURL is the config for the presigned url middleware
type PreSignedURL struct {
	AllowedHTTPMethods []string     `yaml:"allowed_http_methods"`
	Enabled            bool         `yaml:"enabled" env:"PROXY_ENABLE_PRESIGNEDURLS" desc:"Allow OCS to get a signing key to sign requests."`
	SigningKeys        *SigningKeys `yaml:"signing_keys"`
}

// SigningKeys is a store configuration.
type SigningKeys struct {
	Store              string        `yaml:"store" env:"OCIS_CACHE_STORE;PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE" desc:"The type of the signing key store. Supported values are: 'redis-sentinel', 'nats-js-kv' and 'ocisstoreservice' (deprecated). See the text description for details."`
	Nodes              []string      `yaml:"addresses" env:"OCIS_CACHE_STORE_NODES;PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_NODES" desc:"A list of nodes to access the configured store. Note that the behaviour how nodes are used is dependent on the library of the configured store. See the Environment Variable Types description for more details."`
	TTL                time.Duration `yaml:"ttl" env:"OCIS_CACHE_TTL;PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_TTL" desc:"Default time to live for signing keys. See the Environment Variable Types description for more details."`
	DisablePersistence bool          `yaml:"disable_persistence" env:"OCIS_CACHE_DISABLE_PERSISTENCE;PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_DISABLE_PERSISTENCE" desc:"Disables persistence of the store. Only applies when store type 'nats-js-kv' is configured. Defaults to true."`
	AuthUsername       string        `yaml:"username" env:"OCIS_CACHE_AUTH_USERNAME;PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_AUTH_USERNAME" desc:"The username to authenticate with the store. Only applies when store type 'nats-js-kv' is configured."`
	AuthPassword       string        `yaml:"password" env:"OCIS_CACHE_AUTH_PASSWORD;PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_AUTH_PASSWORD" desc:"The password to authenticate with the store. Only applies when store type 'nats-js-kv' is configured."`
}

// ClaimsSelectorConf is the config for the claims-selector
type ClaimsSelectorConf struct {
	DefaultPolicy         string `yaml:"default_policy"`
	UnauthenticatedPolicy string `yaml:"unauthenticated_policy"`
	SelectorCookieName    string `yaml:"selector_cookie_name"`
}

// RegexSelectorConf is the config for the regex-selector
type RegexSelectorConf struct {
	DefaultPolicy         string          `yaml:"default_policy"`
	MatchesPolicies       []RegexRuleConf `yaml:"matches_policies"`
	UnauthenticatedPolicy string          `yaml:"unauthenticated_policy"`
	SelectorCookieName    string          `yaml:"selector_cookie_name"`
}

type RegexRuleConf struct {
	Priority int    `yaml:"priority"`
	Property string `yaml:"property"`
	Match    string `yaml:"match"`
	Policy   string `yaml:"policy"`
}

// ServiceAccount is the configuration for the used service account
type ServiceAccount struct {
	ServiceAccountID     string `yaml:"service_account_id" env:"OCIS_SERVICE_ACCOUNT_ID;PROXY_SERVICE_ACCOUNT_ID" desc:"The ID of the service account the service should use. See the 'auth-service' service description for more details."`
	ServiceAccountSecret string `yaml:"service_account_secret" env:"OCIS_SERVICE_ACCOUNT_SECRET;PROXY_SERVICE_ACCOUNT_SECRET" desc:"The service account secret."`
}
