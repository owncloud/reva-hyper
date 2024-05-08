package errors

import (
	"encoding/xml"

	"github.com/pkg/errors"
)

// Exception represents a ocdav exception
type Exception struct {
	Code    int
	Message string
	Header  string
}

// ErrorXML holds the xml representation of an error
// http://www.webdav.org/specs/rfc4918.html#ELEMENT_error
type ErrorXML struct {
	XMLName   xml.Name `xml:"d:error"`
	Xmlnsd    string   `xml:"xmlns:d,attr"`
	Xmlnss    string   `xml:"xmlns:s,attr"`
	Exception string   `xml:"s:exception"`
	Message   string   `xml:"s:message"`
	InnerXML  []byte   `xml:",innerxml"`
	// Header is used to indicate the conflicting request header
	Header string `xml:"s:header,omitempty"`
}

var (
	// ErrInvalidPropfind is an invalid propfind error
	ErrInvalidPropfind = errors.New("webdav: invalid propfind")
)
