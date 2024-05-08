package propfind

import (
	"fmt"

	"encoding/xml"

	"github.com/owncloud/ocis/v2/services/webdav/pkg/errors"
	"github.com/owncloud/ocis/v2/services/webdav/pkg/prop"
)

// Props represents properties related to a resource
// http://www.webdav.org/specs/rfc4918.html#ELEMENT_prop (for propfind)
type Props []xml.Name

// XML holds the xml representation of a propfind
// http://www.webdav.org/specs/rfc4918.html#ELEMENT_propfind
type XML struct {
	XMLName  xml.Name  `xml:"DAV: propfind"`
	AllProp  *struct{} `xml:"DAV: allprop"`
	PropName *struct{} `xml:"DAV: propname"`
	Prop     Props     `xml:"DAV: prop"`
	Include  Props     `xml:"DAV: include"`
}

// PropStatXML holds the xml representation of a propfind response
// http://www.webdav.org/specs/rfc4918.html#ELEMENT_propstat
type PropStatXML struct {
	// Prop requires DAV: to be the default namespace in the enclosing
	// XML. This is due to the standard encoding/xml package currently
	// not honoring namespace declarations inside a xmltag with a
	// parent element for anonymous slice elements.
	// Use of multistatusWriter takes care of this.
	Prop                []prop.PropertyXML `xml:"d:prop>_ignored_"`
	Status              string             `xml:"d:status"`
	Error               *errors.ErrorXML   `xml:"d:error"`
	ResponseDescription string             `xml:"d:responsedescription,omitempty"`
}

// ResponseXML holds the xml representation of a propfind response
type ResponseXML struct {
	XMLName             xml.Name         `xml:"d:response"`
	Href                string           `xml:"d:href"`
	PropStat            []PropStatXML    `xml:"d:propstat"`
	Status              string           `xml:"d:status,omitempty"`
	Error               *errors.ErrorXML `xml:"d:error"`
	ResponseDescription string           `xml:"d:responsedescription,omitempty"`
}

// MultiStatusResponseXML holds the xml representation of a multi-status propfind response
type MultiStatusResponseXML struct {
	XMLName xml.Name `xml:"d:multistatus"`
	XmlnsS  string   `xml:"xmlns:s,attr,omitempty"`
	XmlnsD  string   `xml:"xmlns:d,attr,omitempty"`
	XmlnsOC string   `xml:"xmlns:oc,attr,omitempty"`

	Responses []*ResponseXML `xml:"d:response"`
}

// ResponseUnmarshalXML is a workaround for https://github.com/golang/go/issues/13400
type ResponseUnmarshalXML struct {
	XMLName             xml.Name               `xml:"response"`
	Href                string                 `xml:"href"`
	PropStat            []PropStatUnmarshalXML `xml:"propstat"`
	Status              string                 `xml:"status,omitempty"`
	Error               *errors.ErrorXML       `xml:"d:error"`
	ResponseDescription string                 `xml:"responsedescription,omitempty"`
}

// MultiStatusResponseUnmarshalXML is a workaround for https://github.com/golang/go/issues/13400
type MultiStatusResponseUnmarshalXML struct {
	XMLName xml.Name `xml:"multistatus"`
	XmlnsS  string   `xml:"xmlns:s,attr,omitempty"`
	XmlnsD  string   `xml:"xmlns:d,attr,omitempty"`
	XmlnsOC string   `xml:"xmlns:oc,attr,omitempty"`

	Responses []*ResponseUnmarshalXML `xml:"response"`
}

// PropStatUnmarshalXML is a workaround for https://github.com/golang/go/issues/13400
type PropStatUnmarshalXML struct {
	// Prop requires DAV: to be the default namespace in the enclosing
	// XML. This is due to the standard encoding/xml package currently
	// not honoring namespace declarations inside a xmltag with a
	// parent element for anonymous slice elements.
	// Use of multistatusWriter takes care of this.
	Prop                []*prop.PropertyXML `xml:"prop"`
	Status              string              `xml:"status"`
	Error               *errors.ErrorXML    `xml:"d:error"`
	ResponseDescription string              `xml:"responsedescription,omitempty"`
}

// NewMultiStatusResponseXML returns a preconfigured instance of MultiStatusResponseXML
func NewMultiStatusResponseXML() *MultiStatusResponseXML {
	return &MultiStatusResponseXML{
		XmlnsD:  "DAV:",
		XmlnsS:  "http://sabredav.org/ns",
		XmlnsOC: "http://owncloud.org/ns",
	}
}

// UnmarshalXML appends the property names enclosed within start to pn.
//
// It returns an error if start does not contain any properties or if
// properties contain values. Character data between properties is ignored.
func (pn *Props) UnmarshalXML(d *xml.Decoder, _ xml.StartElement) error {
	for {
		t, err := prop.Next(d)
		if err != nil {
			return err
		}
		switch e := t.(type) {
		case xml.EndElement:
			// jfd: I think <d:prop></d:prop> is perfectly valid ... treat it as allprop
			/*
				if len(*pn) == 0 {
					return fmt.Errorf("%s must not be empty", start.Name.Local)
				}
			*/
			return nil
		case xml.StartElement:
			t, err = prop.Next(d)
			if err != nil {
				return err
			}
			if _, ok := t.(xml.EndElement); !ok {
				return fmt.Errorf("unexpected token %T", t)
			}
			*pn = append(*pn, e.Name)
		}
	}
}
