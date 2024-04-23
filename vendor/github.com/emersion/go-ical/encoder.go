package ical

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
)

func checkComponent(comp *Component) error {
	var exactlyOneProps, atMostOneProps []string
	switch comp.Name {
	case CompCalendar:
		if len(comp.Children) == 0 {
			return fmt.Errorf("ical: failed to encode VCALENDAR: calendar is empty")
		}

		exactlyOneProps = []string{PropProductID, PropVersion}
		atMostOneProps = []string{
			PropCalendarScale,
			PropMethod,
			PropUID,
			PropLastModified,
			PropURL,
			PropRefreshInterval,
			PropSource,
			PropColor,
		}
	case CompEvent:
		for _, child := range comp.Children {
			if child.Name != CompAlarm {
				return fmt.Errorf("ical: failed to encode VEVENT: nested %q components are forbidden, only VALARM is allowed", child.Name)
			}
		}

		exactlyOneProps = []string{PropDateTimeStamp, PropUID}
		atMostOneProps = []string{
			PropDateTimeStart,
			PropClass,
			PropCreated,
			PropDescription,
			PropGeo,
			PropLastModified,
			PropLocation,
			PropOrganizer,
			PropPriority,
			PropRecurrenceRule,
			PropSequence,
			PropStatus,
			PropSummary,
			PropTransparency,
			PropURL,
			PropRecurrenceID,
			PropDateTimeEnd,
			PropDuration,
			PropColor,
		}

		// TODO: DTSTART is required if VCALENDAR is missing the METHOD prop
		if len(comp.Props[PropDateTimeEnd]) > 0 && len(comp.Props[PropDuration]) > 0 {
			return fmt.Errorf("ical: failed to encode VEVENT: only one of DTEND and DURATION can be specified")
		}
	case CompToDo:
		for _, child := range comp.Children {
			if child.Name != CompAlarm {
				return fmt.Errorf("ical: failed to encode VTODO: nested %q components are forbidden, only VALARM is allowed", child.Name)
			}
		}

		exactlyOneProps = []string{PropDateTimeStamp, PropUID}
		atMostOneProps = []string{
			PropClass,
			PropCompleted,
			PropCreated,
			PropDescription,
			PropDateTimeStart,
			PropGeo,
			PropLastModified,
			PropLocation,
			PropOrganizer,
			PropPercentComplete,
			PropPriority,
			PropRecurrenceID,
			PropSequence,
			PropStatus,
			PropSummary,
			PropURL,
			PropDue,
			PropDuration,
			PropColor,
		}

		if len(comp.Props[PropDue]) > 0 && len(comp.Props[PropDuration]) > 0 {
			return fmt.Errorf("ical: failed to encode VTODO: only one of DUE and DURATION can be specified")
		}
		if len(comp.Props[PropDuration]) > 0 && len(comp.Props[PropDateTimeStart]) == 0 {
			return fmt.Errorf("ical: failed to encode VTODO: DTSTART is required when DURATION is specified")
		}
	case CompJournal:
		exactlyOneProps = []string{PropDateTimeStamp, PropUID}
		atMostOneProps = []string{
			PropClass,
			PropCreated,
			PropDateTimeStart,
			PropLastModified,
			PropOrganizer,
			PropRecurrenceID,
			PropSequence,
			PropStatus,
			PropSummary,
			PropURL,
			PropColor,
		}

		if len(comp.Children) > 0 {
			return fmt.Errorf("ical: failed to encode VJOURNAL: nested components are forbidden")
		}
	case CompFreeBusy:
		exactlyOneProps = []string{PropDateTimeStamp, PropUID}
		atMostOneProps = []string{
			PropContact,
			PropDateTimeStart,
			PropDateTimeEnd,
			PropOrganizer,
			PropURL,
		}

		if len(comp.Children) > 0 {
			return fmt.Errorf("ical: failed to encode VFREEBUSY: nested components are forbidden")
		}
	case CompTimezone:
		if len(comp.Children) == 0 {
			return fmt.Errorf("ical: failed to encode VTIMEZONE: expected one nested STANDARD or DAYLIGHT component")
		}
		for _, child := range comp.Children {
			if child.Name != CompTimezoneStandard && child.Name != CompTimezoneDaylight {
				return fmt.Errorf("ical: failed to encode VTIMEZONE: nested %q components are forbidden, only STANDARD and DAYLIGHT are allowed", child.Name)
			}
		}

		exactlyOneProps = []string{PropTimezoneID}
		atMostOneProps = []string{
			PropLastModified,
			PropTimezoneURL,
		}
	case CompTimezoneStandard, CompTimezoneDaylight:
		exactlyOneProps = []string{
			PropDateTimeStart,
			PropTimezoneOffsetTo,
			PropTimezoneOffsetFrom,
		}
	case CompAlarm:
		// TODO
	}

	for _, name := range exactlyOneProps {
		if n := len(comp.Props[name]); n != 1 {
			return fmt.Errorf("ical: failed to encode %q: want exactly one %q property, got %v", comp.Name, name, n)
		}
	}
	for _, name := range atMostOneProps {
		if n := len(comp.Props[name]); n > 1 {
			return fmt.Errorf("ical: failed to encode %q: want at most one %q property, got %v", comp.Name, name, n)
		}
	}

	return nil
}

type Encoder struct {
	w io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w}
}

func (enc *Encoder) encodeProp(prop *Prop) error {
	var buf bytes.Buffer
	buf.WriteString(prop.Name)

	paramNames := make([]string, 0, len(prop.Params))
	for name := range prop.Params {
		paramNames = append(paramNames, name)
	}
	sort.Strings(paramNames)

	for _, name := range paramNames {
		buf.WriteString(";")
		buf.WriteString(name)
		buf.WriteString("=")

		for i, v := range prop.Params[name] {
			if i > 0 {
				buf.WriteString(",")
			}
			if strings.ContainsRune(v, '"') {
				return fmt.Errorf("ical: failed to encode param value: contains a double-quote")
			}
			if strings.ContainsAny(v, ";:,") {
				buf.WriteString(`"` + v + `"`)
			} else {
				buf.WriteString(v)
			}
		}
	}

	buf.WriteString(":")
	if strings.ContainsAny(prop.Value, "\r\n") {
		return fmt.Errorf("ical: failed to encode property value: contains a CR or LF")
	}
	buf.WriteString(prop.Value)
	buf.WriteString("\r\n")

	_, err := enc.w.Write(buf.Bytes())
	return err
}

func (enc *Encoder) encodeComponent(comp *Component) error {
	if err := checkComponent(comp); err != nil {
		return err
	}

	err := enc.encodeProp(&Prop{Name: "BEGIN", Value: comp.Name})
	if err != nil {
		return err
	}

	propNames := make([]string, 0, len(comp.Props))
	for name := range comp.Props {
		propNames = append(propNames, name)
	}
	sort.Strings(propNames)

	for _, name := range propNames {
		for _, prop := range comp.Props[name] {
			if err := enc.encodeProp(&prop); err != nil {
				return err
			}
		}
	}

	for _, child := range comp.Children {
		if err := enc.encodeComponent(child); err != nil {
			return err
		}
	}

	return enc.encodeProp(&Prop{Name: "END", Value: comp.Name})
}

func (enc *Encoder) Encode(cal *Calendar) error {
	return enc.encodeComponent(cal.Component)
}
