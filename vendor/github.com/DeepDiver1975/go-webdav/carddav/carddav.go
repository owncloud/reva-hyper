// Package carddav provides a client and server CardDAV implementation.
//
// CardDAV is defined in RFC 6352.
package carddav

import (
	"time"

	"github.com/DeepDiver1975/go-webdav"
	"github.com/DeepDiver1975/go-webdav/internal"
	"github.com/emersion/go-vcard"
)

var CapabilityAddressBook = webdav.Capability("addressbook")

func NewAddressBookHomeSet(path string) webdav.BackendSuppliedHomeSet {
	return &addressbookHomeSet{Href: internal.Href{Path: path}}
}

type AddressDataType struct {
	ContentType string
	Version     string
}

type AddressBook struct {
	Path                 string
	Name                 string
	Description          string
	MaxResourceSize      int64
	SupportedAddressData []AddressDataType
}

func (ab *AddressBook) SupportsAddressData(contentType, version string) bool {
	if len(ab.SupportedAddressData) == 0 {
		return contentType == "text/vcard" && version == "3.0"
	}
	for _, t := range ab.SupportedAddressData {
		if t.ContentType == contentType && t.Version == version {
			return true
		}
	}
	return false
}

type AddressBookQuery struct {
	DataRequest AddressDataRequest

	PropFilters []PropFilter
	FilterTest  FilterTest // defaults to FilterAnyOf

	Limit int // <= 0 means unlimited
}

type AddressDataRequest struct {
	Props   []string
	AllProp bool
}

type PropFilter struct {
	Name string
	Test FilterTest // defaults to FilterAnyOf

	// if IsNotDefined is set, TextMatches and Params need to be unset
	IsNotDefined bool
	TextMatches  []TextMatch
	Params       []ParamFilter
}

type ParamFilter struct {
	Name string

	// if IsNotDefined is set, TextMatch needs to be unset
	IsNotDefined bool
	TextMatch    *TextMatch
}

type TextMatch struct {
	Text            string
	NegateCondition bool
	MatchType       MatchType // defaults to MatchContains
}

type FilterTest string

const (
	FilterAnyOf FilterTest = "anyof"
	FilterAllOf FilterTest = "allof"
)

type MatchType string

const (
	MatchEquals     MatchType = "equals"
	MatchContains   MatchType = "contains"
	MatchStartsWith MatchType = "starts-with"
	MatchEndsWith   MatchType = "ends-with"
)

type AddressBookMultiGet struct {
	Paths       []string
	DataRequest AddressDataRequest
}

type AddressObject struct {
	Path          string
	ModTime       time.Time
	ContentLength int64
	ETag          string
	Card          vcard.Card
}

// SyncQuery is the query struct represents a sync-collection request
type SyncQuery struct {
	DataRequest AddressDataRequest
	SyncToken   string
	Limit       int // <= 0 means unlimited
}

// SyncResponse contains the returned sync-token for next time
type SyncResponse struct {
	SyncToken string
	Updated   []AddressObject
	Deleted   []string
}
