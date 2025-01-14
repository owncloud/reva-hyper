package oidc

import (
	"strings"
)

const (
	Iss               = "iss"
	Sub               = "sub"
	Email             = "email"
	Name              = "name"
	PreferredUsername = "preferred_username"
	UIDNumber         = "uidnumber"
	GIDNumber         = "gidnumber"
	Groups            = "groups"
	OwncloudUUID      = "ownclouduuid"
	OcisRoutingPolicy = "ocis.routing.policy"
)

// SplitWithEscaping splits s into segments using separator which can be escaped using the escape string
// See https://codereview.stackexchange.com/a/280193
func SplitWithEscaping(s string, separator string, escapeString string) []string {
	a := strings.Split(s, separator)

	for i := len(a) - 2; i >= 0; i-- {
		if strings.HasSuffix(a[i], escapeString) {
			a[i] = a[i][:len(a[i])-len(escapeString)] + separator + a[i+1]
			a = append(a[:i+1], a[i+2:]...)
		}
	}
	return a
}
