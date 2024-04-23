// Package ical implements the iCalendar file format.
//
// iCalendar is defined in RFC 5545.
package ical

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

// MIME type and file extension for iCal, defined in RFC 5545 section 8.1.
const (
	MIMEType  = "text/calendar"
	Extension = "ics"
)

const (
	dateFormat        = "20060102"
	datetimeFormat    = "20060102T150405"
	datetimeUTCFormat = "20060102T150405Z"
)

// Params is a set of property parameters.
type Params map[string][]string

func (params Params) Values(name string) []string {
	return params[strings.ToUpper(name)]
}

func (params Params) Get(name string) string {
	if values := params[strings.ToUpper(name)]; len(values) > 0 {
		return values[0]
	}
	return ""
}

func (params Params) Set(name, value string) {
	params[strings.ToUpper(name)] = []string{value}
}

func (params Params) Add(name, value string) {
	name = strings.ToUpper(name)
	params[name] = append(params[name], value)
}

func (params Params) Del(name string) {
	delete(params, strings.ToUpper(name))
}

// Prop is a component property.
type Prop struct {
	Name   string
	Params Params
	Value  string
}

// NewProp creates a new property with the specified name.
func NewProp(name string) *Prop {
	return &Prop{
		Name:   strings.ToUpper(name),
		Params: make(Params),
	}
}

func (prop *Prop) ValueType() ValueType {
	t := ValueType(prop.Params.Get(ParamValue))
	if t == ValueDefault {
		t = defaultValueTypes[prop.Name]
	}
	return t
}

func (prop *Prop) SetValueType(t ValueType) {
	dt, ok := defaultValueTypes[prop.Name]
	if t == ValueDefault || (ok && t == dt) {
		prop.Params.Del(ParamValue)
	} else {
		prop.Params.Set(ParamValue, string(t))
	}
}

func (prop *Prop) expectValueType(want ValueType) error {
	t := prop.ValueType()
	if t != ValueDefault && t != want {
		return fmt.Errorf("ical: property %q: expected type %q, got %q", prop.Name, want, t)
	}
	return nil
}

func (prop *Prop) Binary() ([]byte, error) {
	if err := prop.expectValueType(ValueBinary); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(prop.Value)
}

func (prop *Prop) SetBinary(b []byte) {
	prop.SetValueType(ValueBinary)
	prop.Params.Set("ENCODING", "BASE64")
	prop.Value = base64.StdEncoding.EncodeToString(b)
}

func (prop *Prop) Bool() (bool, error) {
	if err := prop.expectValueType(ValueBool); err != nil {
		return false, err
	}
	switch strings.ToUpper(prop.Value) {
	case "TRUE":
		return true, nil
	case "FALSE":
		return false, nil
	default:
		return false, fmt.Errorf("ical: invalid boolean: %q", prop.Value)
	}
}

// DateTime parses the property value as a date-time or a date.
func (prop *Prop) DateTime(loc *time.Location) (time.Time, error) {
	// Default to UTC, if there is no given location.
	if loc == nil {
		loc = time.UTC
	}

	valueType := prop.ValueType()
	valueLength := len(prop.Value)
	if valueType == ValueDefault {
		switch valueLength {
		case len(dateFormat):
			valueType = ValueDate
		case len(datetimeFormat), len(datetimeUTCFormat):
			valueType = ValueDateTime
		}
	}

	switch valueType {
	case ValueDate:
		return time.ParseInLocation(dateFormat, prop.Value, loc)
	case ValueDateTime:
		if valueLength == len(datetimeUTCFormat) {
			return time.ParseInLocation(datetimeUTCFormat, prop.Value, time.UTC)
		}
		// Use the TZID location, if available.
		if tzid := prop.Params.Get(PropTimezoneID); tzid != "" {
			tzLoc, err := time.LoadLocation(tzid)
			if err != nil {
				return time.Time{}, err
			}
			loc = tzLoc
		}
		return time.ParseInLocation(datetimeFormat, prop.Value, loc)
	}

	return time.Time{}, fmt.Errorf("ical: cannot process: (%q) %s", valueType, prop.Value)
}

func (prop *Prop) SetDate(t time.Time) {
	prop.SetValueType(ValueDate)
	prop.Value = t.Format(dateFormat)
}

func (prop *Prop) SetDateTime(t time.Time) {
	prop.SetValueType(ValueDateTime)
	switch t.Location() {
	case nil, time.UTC:
		prop.Value = t.Format(datetimeUTCFormat)
	default:
		prop.Params.Set(PropTimezoneID, t.Location().String())
		prop.Value = t.Format(datetimeFormat)
	}
}

type durationParser struct {
	s string
}

func (p *durationParser) consume(c byte) bool {
	if len(p.s) == 0 || p.s[0] != c {
		return false
	}
	p.s = p.s[1:]
	return true
}

func (p *durationParser) parseCount() (time.Duration, error) {
	// Find the first non-digit
	i := strings.IndexFunc(p.s, func(r rune) bool {
		return r < '0' || r > '9'
	})
	if i == 0 {
		return 0, fmt.Errorf("ical: invalid duration: expected a digit")
	}
	if i < 0 {
		i = len(p.s)
	}

	n, err := strconv.ParseUint(p.s[:i], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("ical: invalid duration: %v", err)
	}
	p.s = p.s[i:]
	return time.Duration(n), nil
}

func (p *durationParser) parseDuration() (time.Duration, error) {
	neg := p.consume('-')
	if !neg {
		_ = p.consume('+')
	}

	if !p.consume('P') {
		return 0, fmt.Errorf("ical: invalid duration: expected 'P'")
	}

	var dur time.Duration
	isTime := false
	for len(p.s) > 0 {
		if p.consume('T') {
			isTime = true
		}

		n, err := p.parseCount()
		if err != nil {
			return 0, err
		}

		if !isTime {
			if p.consume('D') {
				dur += n * 24 * time.Hour
			} else if p.consume('W') {
				dur += n * 7 * 24 * time.Hour
			} else {
				return 0, fmt.Errorf("ical: invalid duration: expected 'D' or 'W'")
			}
		} else {
			if p.consume('H') {
				dur += n * time.Hour
			} else if p.consume('M') {
				dur += n * time.Minute
			} else if p.consume('S') {
				dur += n * time.Second
			} else {
				return 0, fmt.Errorf("ical: invalid duration: expected 'H', 'M' or 'S'")
			}
		}
	}

	if neg {
		dur = -dur
	}
	return dur, nil
}

func (prop *Prop) Duration() (time.Duration, error) {
	if err := prop.expectValueType(ValueDuration); err != nil {
		return 0, err
	}
	p := durationParser{strings.ToUpper(prop.Value)}
	return p.parseDuration()
}

func (prop *Prop) SetDuration(dur time.Duration) {
	prop.SetValueType(ValueDuration)

	sec := dur.Milliseconds() / 1000
	neg := sec < 0
	if sec < 0 {
		sec = -sec
	}

	var s string
	if neg {
		s += "-"
	}
	s += "PT"
	s += strconv.FormatInt(sec, 10)
	s += "S"

	prop.Value = s
}

func (prop *Prop) Float() (float64, error) {
	if err := prop.expectValueType(ValueFloat); err != nil {
		return 0, err
	}
	return strconv.ParseFloat(prop.Value, 64)
}

func (prop *Prop) Int() (int, error) {
	if err := prop.expectValueType(ValueInt); err != nil {
		return 0, err
	}
	return strconv.Atoi(prop.Value)
}

func (prop *Prop) TextList() ([]string, error) {
	if err := prop.expectValueType(ValueText); err != nil {
		return nil, err
	}

	var l []string
	var sb strings.Builder
	for i := 0; i < len(prop.Value); i++ {
		switch c := prop.Value[i]; c {
		case '\\':
			i++
			if i >= len(prop.Value) {
				return nil, fmt.Errorf("ical: malformed text: antislash at end of text")
			}
			switch c := prop.Value[i]; c {
			case '\\', ';', ',':
				sb.WriteByte(c)
			case 'n', 'N':
				sb.WriteByte('\n')
			default:
				return nil, fmt.Errorf("ical: malformed text: invalid escape sequence '\\%v'", c)
			}
		case ',':
			l = append(l, sb.String())
			sb.Reset()
		default:
			sb.WriteByte(c)
		}
	}
	l = append(l, sb.String())

	return l, nil
}

func (prop *Prop) SetTextList(l []string) {
	prop.SetValueType(ValueText)

	var sb strings.Builder
	for i, text := range l {
		if i > 0 {
			sb.WriteByte(',')
		}

		sb.Grow(len(text))
		for _, r := range text {
			switch r {
			case '\\', ';', ',':
				sb.WriteByte('\\')
				sb.WriteRune(r)
			case '\n':
				sb.WriteString("\\n")
			default:
				sb.WriteRune(r)
			}
		}
	}
	prop.Value = sb.String()
}

func (prop *Prop) Text() (string, error) {
	l, err := prop.TextList()
	if err != nil {
		return "", err
	}
	if len(l) == 0 {
		return "", nil
	}
	return l[0], nil
}

func (prop *Prop) SetText(text string) {
	prop.SetTextList([]string{text})
}

// URI parses the property value as a URI or binary. If the value is binary, a
// data URI is returned.
func (prop *Prop) URI() (*url.URL, error) {
	switch t := prop.ValueType(); t {
	case ValueDefault, ValueURI:
		return url.Parse(prop.Value)
	case ValueBinary:
		mediaType := prop.Params.Get(ParamFormatType)
		return &url.URL{
			Scheme: "data",
			Opaque: mediaType + ";base64," + prop.Value,
		}, nil
	default:
		return nil, fmt.Errorf("ical: expected URI or BINARY, got %q", t)
	}
}

func (prop *Prop) SetURI(u *url.URL) {
	prop.SetValueType(ValueURI)
	prop.Value = u.String()
}

// TODO: Period, Time, UTCOffset

// Props is a set of component properties.
type Props map[string][]Prop

func (props Props) Values(name string) []Prop {
	return props[strings.ToUpper(name)]
}

func (props Props) Get(name string) *Prop {
	if l := props[strings.ToUpper(name)]; len(l) > 0 {
		return &l[0]
	}
	return nil
}

func (props Props) Set(prop *Prop) {
	props[prop.Name] = []Prop{*prop}
}

func (props Props) Add(prop *Prop) {
	props[prop.Name] = append(props[prop.Name], *prop)
}

func (props Props) Del(name string) {
	delete(props, name)
}

func (props Props) Text(name string) (string, error) {
	if prop := props.Get(name); prop != nil {
		return prop.Text()
	}
	return "", nil
}

func (props Props) SetText(name, text string) {
	prop := NewProp(name)
	prop.SetText(text)
	props.Set(prop)
}

func (props Props) DateTime(name string, loc *time.Location) (time.Time, error) {
	if prop := props.Get(name); prop != nil {
		return prop.DateTime(loc)
	}
	return time.Time{}, nil
}

func (props Props) SetDate(name string, t time.Time) {
	prop := NewProp(name)
	prop.SetDate(t)
	props.Set(prop)
}

func (props Props) SetDateTime(name string, t time.Time) {
	prop := NewProp(name)
	prop.SetDateTime(t)
	props.Set(prop)
}

func (props Props) SetURI(name string, u *url.URL) {
	prop := NewProp(name)
	prop.SetURI(u)
	props.Set(prop)
}

func (props Props) URI(name string) (*url.URL, error) {
	if prop := props.Get(name); prop != nil {
		return prop.URI()
	}
	return nil, nil
}

// Returns an ROption based on the events RRULE.
//
// This object can then be used to construct `RRule` instances for different
// fields, for example, an rrule based on `DTSTART`:
//
//	roption, err := props.RecurrenceRule()
//	if err != nil {
//		log.Fatalf("error parsing rrule:", err)
//	}
//	if roption == nil {
//		log.Fatalf("props have no RRULE")
//	}
//
//	dtstart, err := props.DateTime("DTSTART", nil)
//	if err != nil {
//		log.Fatalf("error parsing dtstart:", err)
//	}
//	roption.Dtstart = dtstart
//
//	return rrule.NewRRule(*roption)
//
// This object can then be used to calculate the `DTSTART` of all recurrances.
func (props Props) RecurrenceRule() (*rrule.ROption, error) {
	prop := props.Get(PropRecurrenceRule)
	if prop == nil {
		return nil, nil
	}
	if err := prop.expectValueType(ValueRecurrence); err != nil {
		return nil, err
	}

	roption, err := rrule.StrToROption(prop.Value)
	if err != nil {
		return nil, fmt.Errorf("ical: error parsing rrule: %v", err)
	}

	return roption, nil
}

func (props Props) SetRecurrenceRule(rule *rrule.ROption) {
	if rule != nil {
		prop := NewProp(PropRecurrenceRule)
		prop.SetValueType(ValueRecurrence)
		prop.Value = rule.RRuleString()
		props.Set(prop)
	} else {
		props.Del(PropRecurrenceRule)
	}
}

// Component is an iCalendar component: collections of properties that express
// a particular calendar semantic. A components can be an events, a to-do, a
// journal entry, timezone information, free/busy time information, or an
// alarm.
type Component struct {
	Name     string
	Props    Props
	Children []*Component
}

// NewComponent creates a new component with the specified name.
func NewComponent(name string) *Component {
	return &Component{
		Name:  strings.ToUpper(name),
		Props: make(Props),
	}
}
