package relations

import (
	"context"
	"testing"

	"github.com/owncloud/ocis/v2/ocis-pkg/oidc"
	"github.com/owncloud/ocis/v2/services/webfinger/pkg/config"
	"github.com/owncloud/ocis/v2/services/webfinger/pkg/webfinger"
)

func TestOwnCloudInstanceErr(t *testing.T) {
	_, err := OwnCloudInstance([]config.Instance{}, "http://\n\rinvalid", "http://\n\rinvalid/realms/ocis")
	if err == nil {
		t.Errorf("provider did not err on invalid url: %v", err)
	}
	_, err = OwnCloudInstance([]config.Instance{{Regex: "("}}, "http://docis.tld", "http://docis.tld/realms/ocis")
	if err == nil {
		t.Errorf("provider did not err on invalid regex: %v", err)
	}
	_, err = OwnCloudInstance([]config.Instance{{Href: "{{invalid}}ee"}}, "http://docis.tld", "http://docis.tld/realms/ocis")
	if err == nil {
		t.Errorf("provider did not err on invalid href template: %v", err)
	}
}

func TestOwnCloudInstanceAddLink(t *testing.T) {
	provider, err := OwnCloudInstance([]config.Instance{{
		Claim: "customclaim",
		Regex: ".+@.+\\..+",
		Href:  "https://{{.otherclaim}}.domain.tld",
		Titles: map[string]string{
			"foo": "bar",
		},
		Break: true,
	}}, "http://docis.tld", "http://docis.tld/realms/ocis")
	if err != nil {
		t.Error(err)
	}

	ctx := context.Background()
	ctx = oidc.NewContext(ctx, map[string]interface{}{
		"customclaim": "some@fizz.buzz",
		"otherclaim":  "someone",
	})
	jrd := webfinger.JSONResourceDescriptor{}
	provider.Add(ctx, &jrd)

	if len(jrd.Links) != 1 {
		t.Errorf("provider returned wrong number of links: %v, expected 1", len(jrd.Links))
	}
	if jrd.Links[0].Href != "https://someone.domain.tld" {
		t.Errorf("provider returned wrong issuer link href: %v, expected %v", jrd.Links[0].Href, "https://someone.domain.tld")
	}
	if jrd.Links[0].Rel != OwnCloudInstanceRel {
		t.Errorf("provider returned owncloud server instance rel: %v, expected %v", jrd.Links[0].Rel, OwnCloudInstanceRel)
	}
	if len(jrd.Links[0].Titles) != 1 {
		t.Errorf("provider returned wrong number of titles: %v, expected 1", len(jrd.Links[0].Titles))
	}
	if jrd.Links[0].Titles["foo"] != "bar" {
		t.Errorf("provider returned wrong title: %v, expected bar", len(jrd.Links[0].Titles["foo"]))
	}

}
