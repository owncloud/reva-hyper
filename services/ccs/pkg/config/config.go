package config

import (
	"context"

	"github.com/owncloud/ocis/v2/ocis-pkg/shared"
)

type Storage struct {
	GatewayAddress string `yaml:"gateway_addr" env:"STORAGE_GATEWAY_GRPC_ADDR" desc:"GRPC address of the STORAGE-SYSTEM service." introductionVersion:"5.0"`

	SystemUserID     string `yaml:"system_user_id" env:"OCIS_SYSTEM_USER_ID;CCS_SYSTEM_USER_ID" desc:"ID of the oCIS STORAGE-SYSTEM system user. Admins need to set the ID for the STORAGE-SYSTEM system user in this config option which is then used to reference the user. Any reasonable long string is possible, preferably this would be an UUIDv4 format." introductionVersion:"5.0"`
	SystemUserIDP    string `yaml:"system_user_idp" env:"OCIS_SYSTEM_USER_IDP;CCS_SYSTEM_USER_IDP" desc:"IDP of the oCIS STORAGE-SYSTEM system user." introductionVersion:"5.0"`
	SystemUserAPIKey string `yaml:"system_user_api_key" env:"OCIS_SYSTEM_USER_API_KEY" desc:"API key for the STORAGE-SYSTEM system user." introductionVersion:"5.0"`
}

type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service
	Service Service         `yaml:"-"`
	Tracing *Tracing        `yaml:"tracing"`
	Log     *Log            `yaml:"log"`
	Debug   Debug           `yaml:"debug"`
	Storage Storage         `yaml:"storage"`

	HTTP HTTPConfig `yaml:"http"`

	Context   context.Context `yaml:"-"`
	JWTSecret string          `yaml:"jwt_secret" env:"OCIS_JWT_SECRET;CCS_JWT_SECRET" desc:"The secret to mint and validate jwt tokens." introductionVersion:"5.0"`
}

type Log struct {
	Level  string `yaml:"level" env:"OCIS_LOG_LEVEL;CCS_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"5.0"`
	Pretty bool   `yaml:"pretty" env:"OCIS_LOG_PRETTY;CCS_LOG_PRETTY" desc:"Activates pretty log output." introductionVersion:"5.0"`
	Color  bool   `yaml:"color" env:"OCIS_LOG_COLOR;CCS_LOG_COLOR" desc:"Activates colorized log output." introductionVersion:"5.0"`
	File   string `yaml:"file" env:"OCIS_LOG_FILE;CCS_LOG_FILE" desc:"The path to the log file. Activates logging to this file if set." introductionVersion:"5.0"`
}

type Service struct {
	Name string `yaml:"-"`
}

type Debug struct {
	Addr   string `yaml:"addr" env:"CCS_DEBUG_ADDR" desc:"Bind address of the debug server, where metrics, health, config and debug endpoints will be exposed." introductionVersion:"5.0"`
	Token  string `yaml:"token" env:"CCS_DEBUG_TOKEN" desc:"Token to secure the metrics endpoint." introductionVersion:"5.0"`
	Pprof  bool   `yaml:"pprof" env:"CCS_DEBUG_PPROF" desc:"Enables pprof, which can be used for profiling." introductionVersion:"5.0"`
	Zpages bool   `yaml:"zpages" env:"CCS_DEBUG_ZPAGES" desc:"Enables zpages, which can be used for collecting and viewing in-memory traces." introductionVersion:"5.0"`
}

type HTTPConfig struct {
	Addr      string                `yaml:"addr" env:"CCS_HTTP_ADDR" desc:"The bind address of the HTTP service." introductionVersion:"5.0"`
	Namespace string                `yaml:"-"`
	Root      string                `yaml:"root" env:"CCS_HTTP_ROOT" desc:"Subdirectory that serves as the root for this HTTP service." introductionVersion:"5.0"`
	TLS       shared.HTTPServiceTLS `yaml:"tls"`
	Protocol  string                `yaml:"protocol" env:"CCS_HTTP_PROTOCOL" desc:"The transport protocol of the HTTP service." introductionVersion:"5.0"`
	Prefix    string                `yaml:"prefix" env:"CCS_HTTP_PREFIX" desc:"A URL path prefix for the handler." introductionVersion:"5.0"`
	CORS      `yaml:"cors"`
}

// CORS defines the available cors configuration.
type CORS struct {
	AllowedOrigins   []string `yaml:"allow_origins" env:"OCIS_CORS_ALLOW_ORIGINS;CCS_CORS_ALLOW_ORIGINS" desc:"A list of allowed CORS origins. See following chapter for more details: *Access-Control-Allow-Origin* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin. See the Environment Variable Types description for more details." introductionVersion:"5.0"`
	AllowedMethods   []string `yaml:"allow_methods" env:"OCIS_CORS_ALLOW_METHODS;CCS_CORS_ALLOW_METHODS" desc:"A list of allowed CORS methods. See following chapter for more details: *Access-Control-Request-Method* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Request-Method. See the Environment Variable Types description for more details." introductionVersion:"5.0"`
	AllowedHeaders   []string `yaml:"allow_headers" env:"OCIS_CORS_ALLOW_HEADERS;CCS_CORS_ALLOW_HEADERS" desc:"A list of allowed CORS headers. See following chapter for more details: *Access-Control-Request-Headers* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Request-Headers. See the Environment Variable Types description for more details." introductionVersion:"5.0"`
	AllowCredentials bool     `yaml:"allow_credentials" env:"OCIS_CORS_ALLOW_CREDENTIALS;CCS_CORS_ALLOW_CREDENTIALS" desc:"Allow credentials for CORS.See following chapter for more details: *Access-Control-Allow-Credentials* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Credentials." introductionVersion:"5.0"`
}
