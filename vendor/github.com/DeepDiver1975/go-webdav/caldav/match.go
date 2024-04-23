package caldav

import (
	"strings"
	"time"

	"github.com/emersion/go-ical"
)

// Filter returns the filtered list of calendar objects matching the provided query.
// A nil query will return the full list of calendar objects.
func Filter(query *CalendarQuery, cos []CalendarObject) ([]CalendarObject, error) {
	if query == nil {
		// FIXME: should we always return a copy of the provided slice?
		return cos, nil
	}

	var out []CalendarObject
	for _, co := range cos {
		ok, err := Match(query.CompFilter, &co)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		// TODO properties are not currently filtered even if requested
		out = append(out, co)
	}
	return out, nil
}

// Match reports whether the provided CalendarObject matches the query.
func Match(query CompFilter, co *CalendarObject) (matched bool, err error) {
	if co.Data == nil || co.Data.Component == nil {
		panic("request to process empty calendar object")
	}
	return match(query, co.Data.Component)
}

func match(filter CompFilter, comp *ical.Component) (bool, error) {
	if comp.Name != filter.Name {
		return filter.IsNotDefined, nil
	}

	var zeroDate time.Time
	if filter.Start != zeroDate {
		match, err := matchCompTimeRange(filter.Start, filter.End, comp)
		if err != nil {
			return false, err
		}
		if !match {
			return false, nil
		}
	}
	for _, compFilter := range filter.Comps {
		match, err := matchCompFilter(compFilter, comp)
		if err != nil {
			return false, err
		}
		if !match {
			return false, nil
		}
	}
	for _, propFilter := range filter.Props {
		match, err := matchPropFilter(propFilter, comp)
		if err != nil {
			return false, err
		}
		if !match {
			return false, nil
		}
	}
	return true, nil
}

func matchCompFilter(filter CompFilter, comp *ical.Component) (bool, error) {
	var matches []*ical.Component

	for _, child := range comp.Children {
		match, err := match(filter, child)
		if err != nil {
			return false, err
		} else if match {
			matches = append(matches, child)
		}
	}
	if len(matches) == 0 {
		return filter.IsNotDefined, nil
	}
	return true, nil
}

func matchPropFilter(filter PropFilter, comp *ical.Component) (bool, error) {
	// TODO: this only matches first field, there can be multiple
	field := comp.Props.Get(filter.Name)
	if field == nil {
		return filter.IsNotDefined, nil
	}

	for _, paramFilter := range filter.ParamFilter {
		if !matchParamFilter(paramFilter, field) {
			return false, nil
		}
	}

	var zeroDate time.Time
	if filter.Start != zeroDate {
		match, err := matchPropTimeRange(filter.Start, filter.End, field)
		if err != nil {
			return false, err
		}
		if !match {
			return false, nil
		}
	} else if filter.TextMatch != nil {
		if !matchTextMatch(*filter.TextMatch, field.Value) {
			return false, nil
		}
		return true, nil
	}
	// empty prop-filter, property exists
	return true, nil
}

func matchCompTimeRange(start, end time.Time, comp *ical.Component) (bool, error) {
	// See https://datatracker.ietf.org/doc/html/rfc4791#section-9.9

	// evaluate recurring components
	rset, err := comp.RecurrenceSet(start.Location())
	if err != nil {
		return false, err
	}
	if rset != nil {
		// TODO we can only set inclusive to true or false, but really the
		// start time is inclusive while the end time is not :/
		return len(rset.Between(start, end, true)) > 0, nil
	}

	// TODO handle more than just events
	if comp.Name != ical.CompEvent {
		return false, nil
	}
	event := ical.Event{comp}

	eventStart, err := event.DateTimeStart(start.Location())
	if err != nil {
		return false, err
	}
	eventEnd, err := event.DateTimeEnd(end.Location())
	if err != nil {
		return false, err
	}

	// Event starts in time range
	if eventStart.After(start) && (end.IsZero() || eventStart.Before(end)) {
		return true, nil
	}
	// Event ends in time range
	if eventEnd.After(start) && (end.IsZero() || eventEnd.Before(end)) {
		return true, nil
	}
	// Event covers entire time range plus some
	if eventStart.Before(start) && (!end.IsZero() && eventEnd.After(end)) {
		return true, nil
	}
	return false, nil
}

func matchPropTimeRange(start, end time.Time, field *ical.Prop) (bool, error) {
	// See https://datatracker.ietf.org/doc/html/rfc4791#section-9.9

	ptime, err := field.DateTime(start.Location())
	if err != nil {
		return false, err
	}
	if ptime.After(start) && (end.IsZero() || ptime.Before(end)) {
		return true, nil
	}
	return false, nil
}

func matchParamFilter(filter ParamFilter, field *ical.Prop) bool {
	// TODO there can be multiple values
	value := field.Params.Get(filter.Name)
	if value == "" {
		return filter.IsNotDefined
	} else if filter.IsNotDefined {
		return false
	}
	if filter.TextMatch != nil {
		return matchTextMatch(*filter.TextMatch, value)
	}
	return true
}

func matchTextMatch(txt TextMatch, value string) bool {
	// TODO: handle text-match collation attribute
	match := strings.Contains(value, txt.Text)
	if txt.NegateCondition {
		match = !match
	}
	return match
}
