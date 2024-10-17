package registry

import (
	"os"
	"time"
)

const (
	_registryRegisterIntervalEnv = "EXPERIMENTAL_REGISTER_INTERVAL"
	_registryRegisterTTLEnv      = "EXPERIMENTAL_REGISTER_TTL"

	// Note: _defaultRegisterInterval should always be lower than _defaultRegisterTTL
	_defaultRegisterInterval = time.Second * 25
	_defaultRegisterTTL      = time.Second * 30
)

// GetRegisterInterval returns the register interval from the environment.
func GetRegisterInterval() time.Duration {
	d, err := time.ParseDuration(os.Getenv(_registryRegisterIntervalEnv))
	if err != nil {
		return _defaultRegisterInterval
	}
	return d
}

// GetRegisterTTL returns the register TTL from the environment.
func GetRegisterTTL() time.Duration {
	d, err := time.ParseDuration(os.Getenv(_registryRegisterTTLEnv))
	if err != nil {
		return _defaultRegisterTTL
	}
	return d
}
