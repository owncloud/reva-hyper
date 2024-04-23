package carddav

import (
	"encoding/xml"
	"fmt"

	"github.com/DeepDiver1975/go-webdav/internal"
)

const namespace = "urn:ietf:params:xml:ns:carddav"

var (
	addressBookHomeSetName = xml.Name{namespace, "addressbook-home-set"}

	addressBookName            = xml.Name{namespace, "addressbook"}
	addressBookDescriptionName = xml.Name{namespace, "addressbook-description"}
	supportedAddressDataName   = xml.Name{namespace, "supported-address-data"}
	maxResourceSizeName        = xml.Name{namespace, "max-resource-size"}

	addressBookQueryName    = xml.Name{namespace, "addressbook-query"}
	addressBookMultigetName = xml.Name{namespace, "addressbook-multiget"}

	addressDataName = xml.Name{namespace, "address-data"}
)

// https://tools.ietf.org/html/rfc6352#section-6.2.3
type addressbookHomeSet struct {
	XMLName xml.Name      `xml:"urn:ietf:params:xml:ns:carddav addressbook-home-set"`
	Href    internal.Href `xml:"DAV: href"`
}

func (a *addressbookHomeSet) GetXMLName() xml.Name {
	return addressBookHomeSetName
}

type addressbookDescription struct {
	XMLName     xml.Name `xml:"urn:ietf:params:xml:ns:carddav addressbook-description"`
	Description string   `xml:",chardata"`
}

// https://tools.ietf.org/html/rfc6352#section-6.2.2
type supportedAddressData struct {
	XMLName xml.Name          `xml:"urn:ietf:params:xml:ns:carddav supported-address-data"`
	Types   []addressDataType `xml:"address-data-type"`
}

type addressDataType struct {
	XMLName     xml.Name `xml:"urn:ietf:params:xml:ns:carddav address-data-type"`
	ContentType string   `xml:"content-type,attr"`
	Version     string   `xml:"version,attr"`
}

// https://tools.ietf.org/html/rfc6352#section-6.2.3
type maxResourceSize struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:carddav max-resource-size"`
	Size    int64    `xml:",chardata"`
}

// https://tools.ietf.org/html/rfc6352#section-10.3
type addressbookQuery struct {
	XMLName  xml.Name       `xml:"urn:ietf:params:xml:ns:carddav addressbook-query"`
	Prop     *internal.Prop `xml:"DAV: prop,omitempty"`
	AllProp  *struct{}      `xml:"DAV: allprop,omitempty"`
	PropName *struct{}      `xml:"DAV: propname,omitempty"`
	Filter   filter         `xml:"filter"`
	Limit    *limit         `xml:"limit,omitempty"`
}

// https://tools.ietf.org/html/rfc6352#section-10.5
type filter struct {
	XMLName xml.Name     `xml:"urn:ietf:params:xml:ns:carddav filter"`
	Test    filterTest   `xml:"test,attr,omitempty"`
	Props   []propFilter `xml:"prop-filter"`
}

type filterTest string

func (ft *filterTest) UnmarshalText(b []byte) error {
	switch FilterTest(b) {
	case FilterAnyOf, FilterAllOf:
		*ft = filterTest(b)
		return nil
	default:
		return fmt.Errorf("carddav: invalid filter test value: %q", string(b))
	}
}

// https://tools.ietf.org/html/rfc6352#section-10.5.1
type propFilter struct {
	XMLName xml.Name   `xml:"urn:ietf:params:xml:ns:carddav prop-filter"`
	Name    string     `xml:"name,attr"`
	Test    filterTest `xml:"test,attr,omitempty"`

	IsNotDefined *struct{}     `xml:"is-not-defined,omitempty"`
	TextMatches  []textMatch   `xml:"text-match,omitempty"`
	Params       []paramFilter `xml:"param-filter,omitempty"`
}

// https://tools.ietf.org/html/rfc6352#section-10.5.4
type textMatch struct {
	XMLName         xml.Name        `xml:"urn:ietf:params:xml:ns:carddav text-match"`
	Text            string          `xml:",chardata"`
	Collation       string          `xml:"collation,attr,omitempty"`
	NegateCondition negateCondition `xml:"negate-condition,attr,omitempty"`
	MatchType       matchType       `xml:"match-type,attr,omitempty"`
}

type negateCondition bool

func (nc *negateCondition) UnmarshalText(b []byte) error {
	switch s := string(b); s {
	case "yes":
		*nc = true
	case "no":
		*nc = false
	default:
		return fmt.Errorf("carddav: invalid negate-condition value: %q", s)
	}
	return nil
}

func (nc negateCondition) MarshalText() ([]byte, error) {
	if nc {
		return []byte("yes"), nil
	}
	return nil, nil
}

type matchType MatchType

func (mt *matchType) UnmarshalText(b []byte) error {
	switch MatchType(b) {
	case MatchEquals, MatchContains, MatchStartsWith, MatchEndsWith:
		*mt = matchType(b)
		return nil
	default:
		return fmt.Errorf("carddav: invalid match type value: %q", string(b))
	}
}

// https://tools.ietf.org/html/rfc6352#section-10.5.2
type paramFilter struct {
	XMLName      xml.Name   `xml:"urn:ietf:params:xml:ns:carddav param-filter"`
	Name         string     `xml:"name,attr"`
	IsNotDefined *struct{}  `xml:"is-not-defined"`
	TextMatch    *textMatch `xml:"text-match"`
}

// https://tools.ietf.org/html/rfc6352#section-10.6
type limit struct {
	XMLName  xml.Name `xml:"urn:ietf:params:xml:ns:carddav limit"`
	NResults uint     `xml:"nresults"`
}

// https://tools.ietf.org/html/rfc6352#section-8.7
type addressbookMultiget struct {
	XMLName  xml.Name        `xml:"urn:ietf:params:xml:ns:carddav addressbook-multiget"`
	Hrefs    []internal.Href `xml:"DAV: href"`
	Prop     *internal.Prop  `xml:"DAV: prop,omitempty"`
	AllProp  *struct{}       `xml:"DAV: allprop,omitempty"`
	PropName *struct{}       `xml:"DAV: propname,omitempty"`
}

func newProp(name string, noValue bool) *internal.RawXMLValue {
	attrs := []xml.Attr{{Name: xml.Name{namespace, "name"}, Value: name}}
	if noValue {
		attrs = append(attrs, xml.Attr{Name: xml.Name{namespace, "novalue"}, Value: "yes"})
	}

	xmlName := xml.Name{namespace, "prop"}
	return internal.NewRawXMLElement(xmlName, attrs, nil)
}

// https://tools.ietf.org/html/rfc6352#section-10.4
type addressDataReq struct {
	XMLName xml.Name  `xml:"urn:ietf:params:xml:ns:carddav address-data"`
	Props   []prop    `xml:"prop"`
	Allprop *struct{} `xml:"allprop"`
}

// https://tools.ietf.org/html/rfc6352#section-10.4.2
type prop struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:carddav prop"`
	Name    string   `xml:"name,attr"`
	// TODO: novalue
}

// https://tools.ietf.org/html/rfc6352#section-10.4
type addressDataResp struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:carddav address-data"`
	Data    []byte   `xml:",chardata"`
}

type reportReq struct {
	Query    *addressbookQuery
	Multiget *addressbookMultiget
}

func (r *reportReq) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var v interface{}
	switch start.Name {
	case addressBookQueryName:
		r.Query = &addressbookQuery{}
		v = r.Query
	case addressBookMultigetName:
		r.Multiget = &addressbookMultiget{}
		v = r.Multiget
	default:
		return fmt.Errorf("carddav: unsupported REPORT root %q %q", start.Name.Space, start.Name.Local)
	}

	return d.DecodeElement(v, &start)
}

type mkcolReq struct {
	XMLName      xml.Name               `xml:"DAV: mkcol"`
	ResourceType internal.ResourceType  `xml:"set>prop>resourcetype"`
	DisplayName  string                 `xml:"set>prop>displayname"`
	Description  addressbookDescription `xml:"set>prop>addressbook-description"`
	// TODO this could theoretically contain all addressbook properties?
}
