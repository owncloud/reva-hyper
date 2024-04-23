package carddav

import (
	"fmt"
	"strings"

	"github.com/emersion/go-vcard"
)

func filterProperties(req AddressDataRequest, ao AddressObject) AddressObject {
	if req.AllProp || len(req.Props) == 0 {
		return ao
	}

	if len(ao.Card) == 0 {
		panic("request to process empty vCard")
	}

	result := AddressObject{
		Path:    ao.Path,
		ModTime: ao.ModTime,
		ETag:    ao.ETag,
	}

	result.Card = make(vcard.Card)
	// result would be invalid w/o version
	result.Card[vcard.FieldVersion] = ao.Card[vcard.FieldVersion]
	for _, prop := range req.Props {
		value, ok := ao.Card[prop]
		if ok {
			result.Card[prop] = value
		}
	}

	return result
}

// Filter returns the filtered list of address objects matching the provided query.
// A nil query will return the full list of address objects.
func Filter(query *AddressBookQuery, aos []AddressObject) ([]AddressObject, error) {
	if query == nil {
		// FIXME: should we always return a copy of the provided slice?
		return aos, nil
	}

	n := query.Limit
	if n <= 0 || n > len(aos) {
		n = len(aos)
	}
	out := make([]AddressObject, 0, n)
	for _, ao := range aos {
		ok, err := Match(query, &ao)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		out = append(out, filterProperties(query.DataRequest, ao))
		if len(out) >= n {
			break
		}
	}
	return out, nil
}

// Match reports whether the provided AddressObject matches the query.
func Match(query *AddressBookQuery, ao *AddressObject) (matched bool, err error) {
	if query == nil {
		return true, nil
	}

	switch query.FilterTest {
	default:
		return false, fmt.Errorf("unknown query filter test %q", query.FilterTest)

	case FilterAnyOf, "":
		for _, prop := range query.PropFilters {
			ok, err := matchPropFilter(prop, ao)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil

	case FilterAllOf:
		for _, prop := range query.PropFilters {
			ok, err := matchPropFilter(prop, ao)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
		return true, nil
	}
}

func matchPropFilter(prop PropFilter, ao *AddressObject) (bool, error) {
	// TODO: this only matches first field, there could be multiple
	field := ao.Card.Get(prop.Name)
	if field == nil {
		return prop.IsNotDefined, nil
	} else if prop.IsNotDefined {
		return false, nil
	}

	// TODO: handle carddav.PropFilter.Params.
	if len(prop.TextMatches) == 0 {
		return true, nil
	}

	switch prop.Test {
	default:
		return false, fmt.Errorf("unknown property filter test %q", prop.Test)

	case FilterAnyOf, "":
		for _, txt := range prop.TextMatches {
			ok, err := matchTextMatch(txt, field)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil

	case FilterAllOf:
		for _, txt := range prop.TextMatches {
			ok, err := matchTextMatch(txt, field)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
		return true, nil
	}
}

func matchTextMatch(txt TextMatch, field *vcard.Field) (bool, error) {
	// TODO: handle text-match collation attribute
	var ok bool
	switch txt.MatchType {
	default:
		return false, fmt.Errorf("unknown textmatch type %q", txt.MatchType)

	case MatchEquals:
		ok = txt.Text == field.Value

	case MatchContains, "":
		ok = strings.Contains(field.Value, txt.Text)

	case MatchStartsWith:
		ok = strings.HasPrefix(field.Value, txt.Text)

	case MatchEndsWith:
		ok = strings.HasSuffix(field.Value, txt.Text)
	}

	if txt.NegateCondition {
		ok = !ok
	}
	return ok, nil
}
