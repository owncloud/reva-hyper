package caldav

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/DeepDiver1975/go-webdav/internal"
)

const namespace = "urn:ietf:params:xml:ns:caldav"

var (
	calendarHomeSetName = xml.Name{namespace, "calendar-home-set"}

	calendarDescriptionName           = xml.Name{namespace, "calendar-description"}
	supportedCalendarDataName         = xml.Name{namespace, "supported-calendar-data"}
	supportedCalendarComponentSetName = xml.Name{namespace, "supported-calendar-component-set"}
	maxResourceSizeName               = xml.Name{namespace, "max-resource-size"}

	calendarQueryName    = xml.Name{namespace, "calendar-query"}
	calendarMultigetName = xml.Name{namespace, "calendar-multiget"}

	calendarName      = xml.Name{namespace, "calendar"}
	calendarDataName  = xml.Name{namespace, "calendar-data"}
	calendarColorName = xml.Name{
		Space: "http://apple.com/ns/ical/",
		Local: "calendar-color",
	}
	calendarTimezoneName = xml.Name{
		Space: namespace,
		Local: "calendar-timezone",
	}
)

// https://tools.ietf.org/html/rfc4791#section-6.2.1
type calendarHomeSet struct {
	XMLName xml.Name      `xml:"urn:ietf:params:xml:ns:caldav calendar-home-set"`
	Href    internal.Href `xml:"DAV: href"`
}

func (a *calendarHomeSet) GetXMLName() xml.Name {
	return calendarHomeSetName
}

// https://tools.ietf.org/html/rfc4791#section-5.2.1
type calendarDescription struct {
	XMLName     xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar-description"`
	Description string   `xml:",chardata"`
}

type calendarColor struct {
	XMLName xml.Name `xml:"http://apple.com/ns/ical/ calendar-color"`
	Color   string   `xml:",chardata"`
}

type calendarTimezone struct {
	XMLName  xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar-timezone"`
	Timezone string   `xml:",chardata"`
}

// https://tools.ietf.org/html/rfc4791#section-5.2.4
type supportedCalendarData struct {
	XMLName xml.Name           `xml:"urn:ietf:params:xml:ns:caldav supported-calendar-data"`
	Types   []calendarDataType `xml:"calendar-data"`
}

// https://tools.ietf.org/html/rfc4791#section-5.2.3
type supportedCalendarComponentSet struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:caldav supported-calendar-component-set"`
	Comp    []comp   `xml:"comp"`
}

// https://tools.ietf.org/html/rfc4791#section-9.6
type calendarDataType struct {
	XMLName     xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
	ContentType string   `xml:"content-type,attr"`
	Version     string   `xml:"version,attr"`
}

// https://tools.ietf.org/html/rfc4791#section-5.2.5
type maxResourceSize struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:caldav max-resource-size"`
	Size    int64    `xml:",chardata"`
}

// https://tools.ietf.org/html/rfc4791#section-9.5
type calendarQuery struct {
	XMLName  xml.Name       `xml:"urn:ietf:params:xml:ns:caldav calendar-query"`
	Prop     *internal.Prop `xml:"DAV: prop,omitempty"`
	AllProp  *struct{}      `xml:"DAV: allprop,omitempty"`
	PropName *struct{}      `xml:"DAV: propname,omitempty"`
	Filter   filter         `xml:"filter"`
	// TODO: timezone
}

// https://tools.ietf.org/html/rfc4791#section-9.10
type calendarMultiget struct {
	XMLName  xml.Name        `xml:"urn:ietf:params:xml:ns:caldav calendar-multiget"`
	Hrefs    []internal.Href `xml:"DAV: href"`
	Prop     *internal.Prop  `xml:"DAV: prop,omitempty"`
	AllProp  *struct{}       `xml:"DAV: allprop,omitempty"`
	PropName *struct{}       `xml:"DAV: propname,omitempty"`
}

// https://tools.ietf.org/html/rfc4791#section-9.7
type filter struct {
	XMLName    xml.Name   `xml:"urn:ietf:params:xml:ns:caldav filter"`
	CompFilter compFilter `xml:"comp-filter"`
}

// https://tools.ietf.org/html/rfc4791#section-9.7.1
type compFilter struct {
	XMLName      xml.Name     `xml:"urn:ietf:params:xml:ns:caldav comp-filter"`
	Name         string       `xml:"name,attr"`
	IsNotDefined *struct{}    `xml:"is-not-defined,omitempty"`
	TimeRange    *timeRange   `xml:"time-range,omitempty"`
	PropFilters  []propFilter `xml:"prop-filter,omitempty"`
	CompFilters  []compFilter `xml:"comp-filter,omitempty"`
}

// https://tools.ietf.org/html/rfc4791#section-9.7.2
type propFilter struct {
	XMLName      xml.Name      `xml:"urn:ietf:params:xml:ns:caldav prop-filter"`
	Name         string        `xml:"name,attr"`
	IsNotDefined *struct{}     `xml:"is-not-defined,omitempty"`
	TimeRange    *timeRange    `xml:"time-range,omitempty"`
	TextMatch    *textMatch    `xml:"text-match,omitempty"`
	ParamFilter  []paramFilter `xml:"param-filter,omitempty"`
}

// https://tools.ietf.org/html/rfc4791#section-9.7.3
type paramFilter struct {
	XMLName      xml.Name   `xml:"urn:ietf:params:xml:ns:caldav param-filter"`
	Name         string     `xml:"name,attr"`
	IsNotDefined *struct{}  `xml:"is-not-defined,omitempty"`
	TextMatch    *textMatch `xml:"text-match,omitempty"`
}

// https://tools.ietf.org/html/rfc4791#section-9.7.5
type textMatch struct {
	XMLName         xml.Name        `xml:"urn:ietf:params:xml:ns:caldav text-match"`
	Text            string          `xml:",chardata"`
	Collation       string          `xml:"collation,attr,omitempty"`
	NegateCondition negateCondition `xml:"negate-condition,attr,omitempty"`
}

type negateCondition bool

func (nc *negateCondition) UnmarshalText(b []byte) error {
	switch s := string(b); s {
	case "yes":
		*nc = true
	case "no":
		*nc = false
	default:
		return fmt.Errorf("caldav: invalid negate-condition value: %q", s)
	}
	return nil
}

func (nc negateCondition) MarshalText() ([]byte, error) {
	if nc {
		return []byte("yes"), nil
	}
	return nil, nil
}

// https://tools.ietf.org/html/rfc4791#section-9.9
type timeRange struct {
	XMLName xml.Name        `xml:"urn:ietf:params:xml:ns:caldav time-range"`
	Start   dateWithUTCTime `xml:"start,attr,omitempty"`
	End     dateWithUTCTime `xml:"end,attr,omitempty"`
}

const dateWithUTCTimeLayout = "20060102T150405Z"

// dateWithUTCTime is the "date with UTC time" format defined in RFC 5545 page
// 34.
type dateWithUTCTime time.Time

func (t *dateWithUTCTime) UnmarshalText(b []byte) error {
	tt, err := time.Parse(dateWithUTCTimeLayout, string(b))
	if err != nil {
		return err
	}
	*t = dateWithUTCTime(tt)
	return nil
}

func (t *dateWithUTCTime) MarshalText() ([]byte, error) {
	s := time.Time(*t).Format(dateWithUTCTimeLayout)
	return []byte(s), nil
}

// Request variant of https://tools.ietf.org/html/rfc4791#section-9.6
type calendarDataReq struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
	Comp    *comp    `xml:"comp,omitempty"`
	// TODO: expand, limit-recurrence-set, limit-freebusy-set
}

// https://tools.ietf.org/html/rfc4791#section-9.6.1
type comp struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:caldav comp"`
	Name    string   `xml:"name,attr"`

	Allprop *struct{} `xml:"allprop,omitempty"`
	Prop    []prop    `xml:"prop,omitempty"`

	Allcomp *struct{} `xml:"allcomp,omitempty"`
	Comp    []comp    `xml:"comp,omitempty"`
}

// https://tools.ietf.org/html/rfc4791#section-9.6.4
type prop struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:caldav prop"`
	Name    string   `xml:"name,attr"`
	// TODO: novalue
}

// Response variant of https://tools.ietf.org/html/rfc4791#section-9.6
type calendarDataResp struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
	Data    []byte   `xml:",chardata"`
}

type reportReq struct {
	Query    *calendarQuery
	Multiget *calendarMultiget
	// TODO: CALDAV:free-busy-query
}

func (r *reportReq) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var v interface{}
	switch start.Name {
	case calendarQueryName:
		r.Query = &calendarQuery{}
		v = r.Query
	case calendarMultigetName:
		r.Multiget = &calendarMultiget{}
		v = r.Multiget
	default:
		return fmt.Errorf("caldav: unsupported REPORT root %q %q", start.Name.Space, start.Name.Local)
	}

	return d.DecodeElement(v, &start)
}

type mkcolReq struct {
	XMLName                       xml.Name                      `xml:"DAV: mkcol"`
	ResourceType                  internal.ResourceType         `xml:"set>prop>resourcetype"`
	DisplayName                   string                        `xml:"set>prop>displayname"`
	Description                   string                        `xml:"set>prop>calendar-description"`
	CalendarColor                 string                        `xml:"set>prop>calendar-color"`
	CalemdarTimeZone              string                        `xml:"set>prop>calendar-timezone"`
	SupportedCalendarComponentSet supportedCalendarComponentSet `xml:"set>prop>supported-calendar-component-set"`
}
