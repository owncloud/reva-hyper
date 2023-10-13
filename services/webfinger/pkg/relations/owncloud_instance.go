package relations

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"text/template"

	"github.com/owncloud/ocis/v2/ocis-pkg/oidc"
	"github.com/owncloud/ocis/v2/services/webfinger/pkg/config"
	"github.com/owncloud/ocis/v2/services/webfinger/pkg/service/v0"
	"github.com/owncloud/ocis/v2/services/webfinger/pkg/webfinger"
)

const (
	OwnCloudInstanceRel = "http://webfinger.owncloud/rel/server-instance"
)

type compiledInstance struct {
	config.Instance
	compiledRegex *regexp.Regexp
	hrefTemplate  *template.Template
}

type ownCloudInstance struct {
	instances    []compiledInstance
	ocisURL      string
	instanceHost string
}

// OwnCloudInstance adds one or more ownCloud instance relations
func OwnCloudInstance(instances []config.Instance, ocisURL string, issuer string) (service.RelationProvider, error) {
	compiledInstances := make([]compiledInstance, 0, len(instances))
	var err error
	for _, instance := range instances {
		compiled := compiledInstance{Instance: instance}
		compiled.compiledRegex, err = regexp.Compile(instance.Regex)
		if err != nil {
			return nil, err
		}
		compiled.hrefTemplate, err = template.New(instance.Claim + ":" + instance.Regex + ":" + instance.Href).Parse(instance.Href)
		if err != nil {
			return nil, err
		}
		compiledInstances = append(compiledInstances, compiled)
	}

	u, err := url.Parse(issuer)
	if err != nil {
		return nil, err
	}
	return &ownCloudInstance{
		instances:    compiledInstances,
		ocisURL:      ocisURL,
		instanceHost: u.Host + u.Path,
	}, nil
}

func (l *ownCloudInstance) Add(ctx context.Context, jrd *webfinger.JSONResourceDescriptor) {
	if jrd == nil {
		jrd = &webfinger.JSONResourceDescriptor{}
	}
	if claims := oidc.FromContext(ctx); claims != nil {
		if value, ok := claims[oidc.PreferredUsername].(string); ok {
			jrd.Subject = "acct:" + value + "@" + l.instanceHost
		} else if value, ok := claims[oidc.Email].(string); ok {
			jrd.Subject = "mailto:" + value
		}
		// allow referencing OCIS_URL in the template
		claims["OCIS_URL"] = l.ocisURL
		for _, instance := range l.instances {
			if value, ok := claims[instance.Claim].(string); ok && instance.compiledRegex.MatchString(value) {
				var tmplWriter strings.Builder
				instance.hrefTemplate.Execute(&tmplWriter, claims)
				jrd.Links = append(jrd.Links, webfinger.Link{
					Rel:    OwnCloudInstanceRel,
					Href:   tmplWriter.String(),
					Titles: instance.Titles,
				})
				if instance.Break {
					break
				}
			}
		}
	}
}
