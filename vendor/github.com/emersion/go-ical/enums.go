package ical

// Components as defined in RFC 5545 section 3.6.
const (
	CompCalendar = "VCALENDAR"
	CompEvent    = "VEVENT"
	CompToDo     = "VTODO"
	CompJournal  = "VJOURNAL"
	CompFreeBusy = "VFREEBUSY"
	CompTimezone = "VTIMEZONE"
	CompAlarm    = "VALARM"
)

// Timezone components.
const (
	CompTimezoneStandard = "STANDARD"
	CompTimezoneDaylight = "DAYLIGHT"
)

// Properties as defined in RFC 5545 section 3.7, RFC 5545 section 3.8 and
// RFC 7986 section 5.
const (
	// Calendar properties
	PropCalendarScale   = "CALSCALE"
	PropMethod          = "METHOD"
	PropProductID       = "PRODID"
	PropVersion         = "VERSION"
	PropName            = "NAME"
	PropRefreshInterval = "REFRESH-INTERVAL"
	PropSource          = "SOURCE"

	// Component properties
	PropAttach          = "ATTACH"
	PropCategories      = "CATEGORIES"
	PropClass           = "CLASS"
	PropComment         = "COMMENT"
	PropDescription     = "DESCRIPTION"
	PropGeo             = "GEO"
	PropLocation        = "LOCATION"
	PropPercentComplete = "PERCENT-COMPLETE"
	PropPriority        = "PRIORITY"
	PropResources       = "RESOURCES"
	PropStatus          = "STATUS"
	PropSummary         = "SUMMARY"
	PropColor           = "COLOR"
	PropImage           = "IMAGE"

	// Date and time component properties
	PropCompleted     = "COMPLETED"
	PropDateTimeEnd   = "DTEND"
	PropDue           = "DUE"
	PropDateTimeStart = "DTSTART"
	PropDuration      = "DURATION"
	PropFreeBusy      = "FREEBUSY"
	PropTransparency  = "TRANSP"

	// Timezone component properties
	PropTimezoneID         = "TZID"
	PropTimezoneName       = "TZNAME"
	PropTimezoneOffsetFrom = "TZOFFSETFROM"
	PropTimezoneOffsetTo   = "TZOFFSETTO"
	PropTimezoneURL        = "TZURL"

	// Relationship component properties
	PropAttendee     = "ATTENDEE"
	PropContact      = "CONTACT"
	PropOrganizer    = "ORGANIZER"
	PropRecurrenceID = "RECURRENCE-ID"
	PropRelatedTo    = "RELATED-TO"
	PropURL          = "URL"
	PropUID          = "UID"
	PropConference   = "CONFERENCE"

	// Recurrence component properties
	PropExceptionDates  = "EXDATE"
	PropRecurrenceDates = "RDATE"
	PropRecurrenceRule  = "RRULE"

	// Alarm component properties
	PropAction  = "ACTION"
	PropRepeat  = "REPEAT"
	PropTrigger = "TRIGGER"

	// Change management component properties
	PropCreated       = "CREATED"
	PropDateTimeStamp = "DTSTAMP"
	PropLastModified  = "LAST-MODIFIED"
	PropSequence      = "SEQUENCE"

	// Miscellaneous component properties
	PropRequestStatus = "REQUEST-STATUS"
)

// Property parameters as defined in RFC 5545 section 3.2 and RFC 7986
// section 6.
const (
	ParamAltRep              = "ALTREP"
	ParamCommonName          = "CN"
	ParamCalendarUserType    = "CUTYPE"
	ParamDelegatedFrom       = "DELEGATED-FROM"
	ParamDelegatedTo         = "DELEGATED-TO"
	ParamDir                 = "DIR"
	ParamEncoding            = "ENCODING"
	ParamFormatType          = "FMTTYPE"
	ParamFreeBusyType        = "FBTYPE"
	ParamLanguage            = "LANGUAGE"
	ParamMember              = "MEMBER"
	ParamParticipationStatus = "PARTSTAT"
	ParamRange               = "RANGE"
	ParamRelated             = "RELATED"
	ParamRelationshipType    = "RELTYPE"
	ParamRole                = "ROLE"
	ParamRSVP                = "RSVP"
	ParamSentBy              = "SENT-BY"
	ParamTimezoneID          = "TZID"
	ParamValue               = "VALUE"
	ParamDisplay             = "DISPLAY"
	ParamEmail               = "EMAIL"
	ParamFeature             = "FEATURE"
	ParamLabel               = "LABEL"
)

// ValueType is the type of a property.
type ValueType string

// Value types as defined in RFC 5545 section 3.3.
const (
	ValueDefault         ValueType = ""
	ValueBinary          ValueType = "BINARY"
	ValueBool            ValueType = "BOOLEAN"
	ValueCalendarAddress ValueType = "CAL-ADDRESS"
	ValueDate            ValueType = "DATE"
	ValueDateTime        ValueType = "DATE-TIME"
	ValueDuration        ValueType = "DURATION"
	ValueFloat           ValueType = "FLOAT"
	ValueInt             ValueType = "INTEGER"
	ValuePeriod          ValueType = "PERIOD"
	ValueRecurrence      ValueType = "RECUR"
	ValueText            ValueType = "TEXT"
	ValueTime            ValueType = "TIME"
	ValueURI             ValueType = "URI"
	ValueUTCOffset       ValueType = "UTC-OFFSET"
)

var defaultValueTypes = map[string]ValueType{
	PropCalendarScale:      ValueText,
	PropMethod:             ValueText,
	PropProductID:          ValueText,
	PropVersion:            ValueText,
	PropAttach:             ValueURI, // can be binary
	PropCategories:         ValueText,
	PropClass:              ValueText,
	PropComment:            ValueText,
	PropDescription:        ValueText,
	PropGeo:                ValueFloat,
	PropLocation:           ValueText,
	PropPercentComplete:    ValueInt,
	PropPriority:           ValueInt,
	PropResources:          ValueText,
	PropStatus:             ValueText,
	PropSummary:            ValueText,
	PropCompleted:          ValueDateTime,
	PropDateTimeEnd:        ValueDateTime, // can be date
	PropDue:                ValueDateTime, // can be date
	PropDateTimeStart:      ValueDateTime, // can be date
	PropDuration:           ValueDuration,
	PropFreeBusy:           ValuePeriod,
	PropTransparency:       ValueText,
	PropTimezoneID:         ValueText,
	PropTimezoneName:       ValueText,
	PropTimezoneOffsetFrom: ValueUTCOffset,
	PropTimezoneOffsetTo:   ValueUTCOffset,
	PropTimezoneURL:        ValueURI,
	PropAttendee:           ValueCalendarAddress,
	PropContact:            ValueText,
	PropOrganizer:          ValueCalendarAddress,
	PropRecurrenceID:       ValueDateTime, // can be date
	PropRelatedTo:          ValueText,
	PropURL:                ValueURI,
	PropUID:                ValueText,
	PropExceptionDates:     ValueDateTime, // can be date
	PropRecurrenceDates:    ValueDateTime, // can be date or period
	PropRecurrenceRule:     ValueRecurrence,
	PropAction:             ValueText,
	PropRepeat:             ValueInt,
	PropTrigger:            ValueDuration, // can be date-time
	PropCreated:            ValueDateTime,
	PropDateTimeStamp:      ValueDateTime,
	PropLastModified:       ValueDateTime,
	PropSequence:           ValueInt,
	PropRequestStatus:      ValueText,
	PropName:               ValueText,
	PropRefreshInterval:    ValueDuration,
	PropSource:             ValueURI,
	PropColor:              ValueText,
	PropImage:              ValueURI, // can be binary
	PropConference:         ValueURI,
}

type EventStatus string

const (
	EventTentative EventStatus = "TENTATIVE"
	EventConfirmed EventStatus = "CONFIRMED"
	EventCancelled EventStatus = "CANCELLED"
)

// ImageDisplay describes the way an image for a component can be displayed.
// Defined in RFC 7986 section 6.1.
type ImageDisplay string

const (
	ImageBadge     ImageDisplay = "BADGE"
	ImageGraphic   ImageDisplay = "GRAPHIC"
	ImageFullSize  ImageDisplay = "FULLSIZE"
	ImageThumbnail ImageDisplay = "THUMBNAIL"
)

// ConferenceFeature describes features of a conference. Defined in RFC 7986
// section 5.7.
type ConferenceFeature string

const (
	ConferenceAudio     ConferenceFeature = "AUDIO"
	ConferenceChat      ConferenceFeature = "CHAT"
	ConferenceFeed      ConferenceFeature = "FEED"
	ConferenceModerator ConferenceFeature = "MODERATOR"
	ConferencePhone     ConferenceFeature = "PHONE"
	ConferenceScreen    ConferenceFeature = "SCREEN"
	ConferenceVideo     ConferenceFeature = "VIDEO"
)
