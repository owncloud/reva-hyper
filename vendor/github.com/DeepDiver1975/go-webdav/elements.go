package webdav

import (
	"encoding/xml"

	"github.com/DeepDiver1975/go-webdav/internal"
)

var (
	principalName                = xml.Name{"DAV:", "principal"}
	principalAlternateURISetName = xml.Name{"DAV:", "alternate-URI-set"}
	principalURLName             = xml.Name{"DAV:", "principal-URL"}
	groupMembershipName          = xml.Name{"DAV:", "group-membership"}
)

// https://datatracker.ietf.org/doc/html/rfc3744#section-4.1
type principalAlternateURISet struct {
	XMLName xml.Name        `xml:"DAV: alternate-URI-set"`
	Hrefs   []internal.Href `xml:"href"`
}

// https://datatracker.ietf.org/doc/html/rfc3744#section-4.2
type principalURL struct {
	XMLName xml.Name      `xml:"DAV: principal-URL"`
	Href    internal.Href `xml:"href"`
}

// https://datatracker.ietf.org/doc/html/rfc3744#section-4.4
type groupMembership struct {
	XMLName xml.Name        `xml:"DAV: group-membership"`
	Hrefs   []internal.Href `xml:"href"`
}
