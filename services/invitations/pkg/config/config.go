package config

import (
	"context"

	"github.com/owncloud/ocis/v2/ocis-pkg/shared"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`

	Tracing *Tracing `yaml:"tracing"`
	Log     *Log     `yaml:"log"`
	Debug   Debug    `yaml:"debug"`

	HTTP HTTP `yaml:"http"`

	Endpoint Endpoint `yaml:"endpoint"`

	TokenManager *TokenManager `yaml:"token_manager"`

	Context context.Context `yaml:"-"`
}

// Endpoint to use
type Endpoint struct {
	URL           string `yaml:"url" env:"INVITATIONS_PROVISIONING_URL" desc:"The endpoint provisioning requests are sent to."`
	Method        string `yaml:"method" env:"INVITATIONS_PROVISIONING_METHOD" desc:"The method to use when making provisioning requests."`
	BodyTemplate  string `yaml:"body_template" env:"INVITATIONS_PROVISIONING_BODY_TEMPLATE" desc:"The template to use as body of a provisioning request."`
	Authorization string `yaml:"authorization" env:"INVITATIONS_PROVISIONING_AUTH" desc:"The authorization to use. Can be 'token' to reuse the access token or 'bearer' to send a static api token."`
	Token         string `yaml:"token" env:"INVITATIONS_PROVISIONING_AUTH_TOKEN" desc:"The bearer token to send in provisioning requests."`
}
