package caldav

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/DeepDiver1975/go-webdav"
	"github.com/DeepDiver1975/go-webdav/internal"
	"github.com/emersion/go-ical"
)

// TODO if nothing more Caldav-specific needs to be added this should be merged with carddav.PutAddressObjectOptions
type PutCalendarObjectOptions struct {
	// IfNoneMatch indicates that the client does not want to overwrite
	// an existing resource.
	IfNoneMatch webdav.ConditionalMatch
	// IfMatch provides the ETag of the resource that the client intends
	// to overwrite, can be ""
	IfMatch webdav.ConditionalMatch
}

// Backend is a CalDAV server backend.
type Backend interface {
	CalendarHomeSetPath(ctx context.Context) (string, error)

	CreateCalendar(ctx context.Context, calendar *Calendar) error
	ListCalendars(ctx context.Context) ([]Calendar, error)
	GetCalendar(ctx context.Context, path string) (*Calendar, error)

	GetCalendarObject(ctx context.Context, path string, req *CalendarCompRequest) (*CalendarObject, error)
	ListCalendarObjects(ctx context.Context, path string, req *CalendarCompRequest) ([]CalendarObject, error)
	QueryCalendarObjects(ctx context.Context, path string, query *CalendarQuery) ([]CalendarObject, error)
	PutCalendarObject(ctx context.Context, path string, calendar *ical.Calendar, opts *PutCalendarObjectOptions) (*CalendarObject, error)
	DeleteCalendarObject(ctx context.Context, path string) error

	webdav.UserPrincipalBackend
}

// Handler handles CalDAV HTTP requests. It can be used to create a CalDAV
// server.
type Handler struct {
	Backend Backend
	Prefix  string
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Backend == nil {
		http.Error(w, "caldav: no backend available", http.StatusInternalServerError)
		return
	}

	if r.URL.Path == "/.well-known/caldav" {
		principalPath, err := h.Backend.CurrentUserPrincipal(r.Context())
		if err != nil {
			http.Error(w, "caldav: failed to determine current user principal", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, principalPath, http.StatusPermanentRedirect)
		return
	}

	var err error
	switch r.Method {
	case "REPORT":
		err = h.handleReport(w, r)
	default:
		b := backend{
			Backend: h.Backend,
			Prefix:  strings.TrimSuffix(h.Prefix, "/"),
		}
		hh := internal.Handler{Backend: &b}
		hh.ServeHTTP(w, r)
	}

	if err != nil {
		internal.ServeError(w, err)
	}
}

func (h *Handler) handleReport(w http.ResponseWriter, r *http.Request) error {
	var report reportReq
	if err := internal.DecodeXMLRequest(r, &report); err != nil {
		return err
	}

	if report.Query != nil {
		return h.handleQuery(r, w, report.Query)
	} else if report.Multiget != nil {
		return h.handleMultiget(r.Context(), w, report.Multiget)
	}
	return internal.HTTPErrorf(http.StatusBadRequest, "caldav: expected calendar-query or calendar-multiget element in REPORT request")
}

func decodeParamFilter(el *paramFilter) (*ParamFilter, error) {
	pf := &ParamFilter{Name: el.Name}
	if el.IsNotDefined != nil {
		if el.TextMatch != nil {
			return nil, fmt.Errorf("caldav: failed to parse param-filter: if is-not-defined is provided, text-match can't be provided")
		}
		pf.IsNotDefined = true
	}
	if el.TextMatch != nil {
		pf.TextMatch = &TextMatch{Text: el.TextMatch.Text}
	}
	return pf, nil
}

func decodePropFilter(el *propFilter) (*PropFilter, error) {
	pf := &PropFilter{Name: el.Name}
	if el.IsNotDefined != nil {
		if el.TextMatch != nil || el.TimeRange != nil || len(el.ParamFilter) > 0 {
			return nil, fmt.Errorf("caldav: failed to parse prop-filter: if is-not-defined is provided, text-match, time-range, or param-filter can't be provided")
		}
		pf.IsNotDefined = true
	}
	if el.TextMatch != nil {
		pf.TextMatch = &TextMatch{Text: el.TextMatch.Text}
	}
	if el.TimeRange != nil {
		pf.Start = time.Time(el.TimeRange.Start)
		pf.End = time.Time(el.TimeRange.End)
	}
	for _, paramEl := range el.ParamFilter {
		paramFi, err := decodeParamFilter(&paramEl)
		if err != nil {
			return nil, err
		}
		pf.ParamFilter = append(pf.ParamFilter, *paramFi)
	}
	return pf, nil
}

func decodeCompFilter(el *compFilter) (*CompFilter, error) {
	cf := &CompFilter{Name: el.Name}
	if el.IsNotDefined != nil {
		if el.TimeRange != nil || len(el.PropFilters) > 0 || len(el.CompFilters) > 0 {
			return nil, fmt.Errorf("caldav: failed to parse comp-filter: if is-not-defined is provided, time-range, prop-filter, or comp-filter can't be provided")
		}
		cf.IsNotDefined = true
	}
	if el.TimeRange != nil {
		cf.Start = time.Time(el.TimeRange.Start)
		cf.End = time.Time(el.TimeRange.End)
	}
	for _, pfEl := range el.PropFilters {
		pf, err := decodePropFilter(&pfEl)
		if err != nil {
			return nil, err
		}
		cf.Props = append(cf.Props, *pf)
	}
	for _, childEl := range el.CompFilters {
		child, err := decodeCompFilter(&childEl)
		if err != nil {
			return nil, err
		}
		cf.Comps = append(cf.Comps, *child)
	}
	return cf, nil
}

func decodeComp(comp *comp) (*CalendarCompRequest, error) {
	if comp == nil {
		return nil, internal.HTTPErrorf(http.StatusBadRequest, "caldav: unexpected empty calendar-data in request")
	}
	if comp.Allprop != nil && len(comp.Prop) > 0 {
		return nil, internal.HTTPErrorf(http.StatusBadRequest, "caldav: only one of allprop or prop can be specified in calendar-data")
	}
	if comp.Allcomp != nil && len(comp.Comp) > 0 {
		return nil, internal.HTTPErrorf(http.StatusBadRequest, "caldav: only one of allcomp or comp can be specified in calendar-data")
	}

	req := &CalendarCompRequest{
		AllProps: comp.Allprop != nil,
		AllComps: comp.Allcomp != nil,
	}
	for _, p := range comp.Prop {
		req.Props = append(req.Props, p.Name)
	}
	for _, c := range comp.Comp {
		comp, err := decodeComp(&c)
		if err != nil {
			return nil, err
		}
		req.Comps = append(req.Comps, *comp)
	}
	return req, nil
}

func decodeCalendarDataReq(calendarData *calendarDataReq) (*CalendarCompRequest, error) {
	if calendarData.Comp == nil {
		return &CalendarCompRequest{
			AllProps: true,
			AllComps: true,
		}, nil
	}
	return decodeComp(calendarData.Comp)
}

func (h *Handler) handleQuery(r *http.Request, w http.ResponseWriter, query *calendarQuery) error {
	var q CalendarQuery
	// TODO: calendar-data in query.Prop
	cf, err := decodeCompFilter(&query.Filter.CompFilter)
	if err != nil {
		return err
	}
	q.CompFilter = *cf

	cos, err := h.Backend.QueryCalendarObjects(r.Context(), r.URL.Path, &q)
	if err != nil {
		return err
	}

	var resps []internal.Response
	for _, co := range cos {
		b := backend{
			Backend: h.Backend,
			Prefix:  strings.TrimSuffix(h.Prefix, "/"),
		}
		propfind := internal.PropFind{
			Prop:     query.Prop,
			AllProp:  query.AllProp,
			PropName: query.PropName,
		}
		resp, err := b.propFindCalendarObject(r.Context(), &propfind, &co)
		if err != nil {
			return err
		}
		resps = append(resps, *resp)
	}

	ms := internal.NewMultiStatus(resps...)

	return internal.ServeMultiStatus(w, ms)
}

func (h *Handler) handleMultiget(ctx context.Context, w http.ResponseWriter, multiget *calendarMultiget) error {
	var dataReq CalendarCompRequest
	if multiget.Prop != nil {
		var calendarData calendarDataReq
		if err := multiget.Prop.Decode(&calendarData); err != nil && !internal.IsNotFound(err) {
			return err
		}
		decoded, err := decodeCalendarDataReq(&calendarData)
		if err != nil {
			return err
		}
		dataReq = *decoded
	}

	var resps []internal.Response
	for _, href := range multiget.Hrefs {
		co, err := h.Backend.GetCalendarObject(ctx, href.Path, &dataReq)
		if err != nil {
			resp := internal.NewErrorResponse(href.Path, err)
			resps = append(resps, *resp)
			continue
		}

		b := backend{
			Backend: h.Backend,
			Prefix:  strings.TrimSuffix(h.Prefix, "/"),
		}
		propfind := internal.PropFind{
			Prop:     multiget.Prop,
			AllProp:  multiget.AllProp,
			PropName: multiget.PropName,
		}
		resp, err := b.propFindCalendarObject(ctx, &propfind, co)
		if err != nil {
			return err
		}
		resps = append(resps, *resp)
	}

	ms := internal.NewMultiStatus(resps...)
	return internal.ServeMultiStatus(w, ms)
}

type backend struct {
	Backend Backend
	Prefix  string
}

type resourceType int

const (
	resourceTypeRoot resourceType = iota
	resourceTypeUserPrincipal
	resourceTypeCalendarHomeSet
	resourceTypeCalendar
	resourceTypeCalendarObject
)

func (b *backend) resourceTypeAtPath(reqPath string) resourceType {
	p := path.Clean(reqPath)
	p = strings.TrimPrefix(p, b.Prefix)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if p == "/" {
		return resourceTypeRoot
	}
	return resourceType(len(strings.Split(p, "/")) - 1)
}

func (b *backend) Options(r *http.Request) (caps []string, allow []string, err error) {
	caps = []string{"calendar-access"}

	if b.resourceTypeAtPath(r.URL.Path) != resourceTypeCalendarObject {
		return caps, []string{http.MethodOptions, "PROPFIND", "REPORT", "DELETE", "MKCOL"}, nil
	}

	var dataReq CalendarCompRequest
	_, err = b.Backend.GetCalendarObject(r.Context(), r.URL.Path, &dataReq)
	if httpErr, ok := err.(*internal.HTTPError); ok && httpErr.Code == http.StatusNotFound {
		return caps, []string{http.MethodOptions, http.MethodPut}, nil
	} else if err != nil {
		return nil, nil, err
	}

	return caps, []string{
		http.MethodOptions,
		http.MethodHead,
		http.MethodGet,
		http.MethodPut,
		http.MethodDelete,
		"PROPFIND",
	}, nil
}

func (b *backend) HeadGet(w http.ResponseWriter, r *http.Request) error {
	var dataReq CalendarCompRequest
	if r.Method != http.MethodHead {
		dataReq.AllProps = true
	}
	co, err := b.Backend.GetCalendarObject(r.Context(), r.URL.Path, &dataReq)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", ical.MIMEType)
	if co.ContentLength > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(co.ContentLength, 10))
	}
	if co.ETag != "" {
		w.Header().Set("ETag", internal.ETag(co.ETag).String())
	}
	if !co.ModTime.IsZero() {
		w.Header().Set("Last-Modified", co.ModTime.UTC().Format(http.TimeFormat))
	}

	if r.Method != http.MethodHead {
		return ical.NewEncoder(w).Encode(co.Data)
	}
	return nil
}

func (b *backend) PropFind(r *http.Request, propfind *internal.PropFind, depth internal.Depth) (*internal.MultiStatus, error) {
	resType := b.resourceTypeAtPath(r.URL.Path)

	var dataReq CalendarCompRequest
	var resps []internal.Response

	switch resType {
	case resourceTypeRoot:
		resp, err := b.propFindRoot(r.Context(), propfind)
		if err != nil {
			return nil, err
		}
		resps = append(resps, *resp)
	case resourceTypeUserPrincipal:
		principalPath, err := b.Backend.CurrentUserPrincipal(r.Context())
		if err != nil {
			return nil, err
		}
		if r.URL.Path == principalPath {
			resp, err := b.propFindUserPrincipal(r.Context(), propfind)
			if err != nil {
				return nil, err
			}
			resps = append(resps, *resp)
			if depth != internal.DepthZero {
				resp, err := b.propFindHomeSet(r.Context(), propfind)
				if err != nil {
					return nil, err
				}
				resps = append(resps, *resp)
				if depth == internal.DepthInfinity {
					resps_, err := b.propFindAllCalendars(r.Context(), propfind, true)
					if err != nil {
						return nil, err
					}
					resps = append(resps, resps_...)
				}
			}
		}
	case resourceTypeCalendarHomeSet:
		homeSetPath, err := b.Backend.CalendarHomeSetPath(r.Context())
		if err != nil {
			return nil, err
		}
		if r.URL.Path == homeSetPath {
			resp, err := b.propFindHomeSet(r.Context(), propfind)
			if err != nil {
				return nil, err
			}
			resps = append(resps, *resp)
			if depth != internal.DepthZero {
				recurse := depth == internal.DepthInfinity
				resps_, err := b.propFindAllCalendars(r.Context(), propfind, recurse)
				if err != nil {
					return nil, err
				}
				resps = append(resps, resps_...)
			}
		}
	case resourceTypeCalendar:
		ab, err := b.Backend.GetCalendar(r.Context(), r.URL.Path)
		if err != nil {
			return nil, err
		}
		resp, err := b.propFindCalendar(r.Context(), propfind, ab)
		if err != nil {
			return nil, err
		}
		resps = append(resps, *resp)
		if depth != internal.DepthZero {
			resps_, err := b.propFindAllCalendarObjects(r.Context(), propfind, ab)
			if err != nil {
				return nil, err
			}
			resps = append(resps, resps_...)
		}
	case resourceTypeCalendarObject:
		ao, err := b.Backend.GetCalendarObject(r.Context(), r.URL.Path, &dataReq)
		if err != nil {
			return nil, err
		}

		resp, err := b.propFindCalendarObject(r.Context(), propfind, ao)
		if err != nil {
			return nil, err
		}
		resps = append(resps, *resp)
	}

	return internal.NewMultiStatus(resps...), nil
}

func (b *backend) propFindRoot(ctx context.Context, propfind *internal.PropFind) (*internal.Response, error) {
	principalPath, err := b.Backend.CurrentUserPrincipal(ctx)
	if err != nil {
		return nil, err
	}

	props := map[xml.Name]internal.PropFindFunc{
		internal.CurrentUserPrincipalName: func(*internal.RawXMLValue) (interface{}, error) {
			return &internal.CurrentUserPrincipal{Href: internal.Href{Path: principalPath}}, nil
		},
		internal.ResourceTypeName: func(*internal.RawXMLValue) (interface{}, error) {
			return internal.NewResourceType(internal.CollectionName), nil
		},
	}
	return internal.NewPropFindResponse(principalPath, propfind, props)
}

func (b *backend) propFindUserPrincipal(ctx context.Context, propfind *internal.PropFind) (*internal.Response, error) {
	principalPath, err := b.Backend.CurrentUserPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	homeSetPath, err := b.Backend.CalendarHomeSetPath(ctx)
	if err != nil {
		return nil, err
	}

	props := map[xml.Name]internal.PropFindFunc{
		internal.CurrentUserPrincipalName: func(*internal.RawXMLValue) (interface{}, error) {
			return &internal.CurrentUserPrincipal{Href: internal.Href{Path: principalPath}}, nil
		},
		calendarHomeSetName: func(*internal.RawXMLValue) (interface{}, error) {
			return &calendarHomeSet{Href: internal.Href{Path: homeSetPath}}, nil
		},
		internal.ResourceTypeName: func(*internal.RawXMLValue) (interface{}, error) {
			return internal.NewResourceType(internal.CollectionName), nil
		},
	}
	return internal.NewPropFindResponse(principalPath, propfind, props)
}

func (b *backend) propFindHomeSet(ctx context.Context, propfind *internal.PropFind) (*internal.Response, error) {
	principalPath, err := b.Backend.CurrentUserPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	homeSetPath, err := b.Backend.CalendarHomeSetPath(ctx)
	if err != nil {
		return nil, err
	}

	// TODO anything else to return here?
	props := map[xml.Name]internal.PropFindFunc{
		internal.CurrentUserPrincipalName: func(*internal.RawXMLValue) (interface{}, error) {
			return &internal.CurrentUserPrincipal{Href: internal.Href{Path: principalPath}}, nil
		},
		internal.ResourceTypeName: func(*internal.RawXMLValue) (interface{}, error) {
			return internal.NewResourceType(internal.CollectionName), nil
		},
	}
	return internal.NewPropFindResponse(homeSetPath, propfind, props)
}

func (b *backend) propFindCalendar(ctx context.Context, propfind *internal.PropFind, cal *Calendar) (*internal.Response, error) {
	props := map[xml.Name]internal.PropFindFunc{
		internal.CurrentUserPrincipalName: func(*internal.RawXMLValue) (interface{}, error) {
			path, err := b.Backend.CurrentUserPrincipal(ctx)
			if err != nil {
				return nil, err
			}
			return &internal.CurrentUserPrincipal{Href: internal.Href{Path: path}}, nil
		},
		internal.ResourceTypeName: func(*internal.RawXMLValue) (interface{}, error) {
			return internal.NewResourceType(internal.CollectionName, calendarName), nil
		},
		calendarDescriptionName: func(*internal.RawXMLValue) (interface{}, error) {
			return &calendarDescription{Description: cal.Description}, nil
		},
		supportedCalendarDataName: func(*internal.RawXMLValue) (interface{}, error) {
			return &supportedCalendarData{
				Types: []calendarDataType{
					{ContentType: ical.MIMEType, Version: "2.0"},
				},
			}, nil
		},
		supportedCalendarComponentSetName: func(*internal.RawXMLValue) (interface{}, error) {
			components := []comp{}
			if cal.SupportedComponentSet != nil {
				for _, name := range cal.SupportedComponentSet {
					components = append(components, comp{Name: name})
				}
			} else {
				components = append(components, comp{Name: ical.CompEvent})
			}
			return &supportedCalendarComponentSet{
				Comp: components,
			}, nil
		},
	}

	if cal.Name != "" {
		props[internal.DisplayNameName] = func(*internal.RawXMLValue) (interface{}, error) {
			return &internal.DisplayName{Name: cal.Name}, nil
		}
	}
	if cal.Description != "" {
		props[calendarDescriptionName] = func(*internal.RawXMLValue) (interface{}, error) {
			return &calendarDescription{Description: cal.Description}, nil
		}
	}
	if cal.Color != "" {
		props[calendarColorName] = func(*internal.RawXMLValue) (interface{}, error) {
			return &calendarColor{
				Color: cal.Color,
			}, nil
		}
	}
	if cal.Timezone != "" {
		props[calendarTimezoneName] = func(*internal.RawXMLValue) (interface{}, error) {
			return &calendarTimezone{
				Timezone: cal.Timezone,
			}, nil
		}
	}
	if cal.MaxResourceSize > 0 {
		props[maxResourceSizeName] = func(*internal.RawXMLValue) (interface{}, error) {
			return &maxResourceSize{Size: cal.MaxResourceSize}, nil
		}
	}

	// TODO: CALDAV:calendar-timezone, CALDAV:supported-calendar-component-set, CALDAV:min-date-time, CALDAV:max-date-time, CALDAV:max-instances, CALDAV:max-attendees-per-instance

	return internal.NewPropFindResponse(cal.Path, propfind, props)
}

func (b *backend) propFindAllCalendars(ctx context.Context, propfind *internal.PropFind, recurse bool) ([]internal.Response, error) {
	abs, err := b.Backend.ListCalendars(ctx)
	if err != nil {
		return nil, err
	}

	var resps []internal.Response
	for _, ab := range abs {
		resp, err := b.propFindCalendar(ctx, propfind, &ab)
		if err != nil {
			return nil, err
		}
		resps = append(resps, *resp)
		if recurse {
			resps_, err := b.propFindAllCalendarObjects(ctx, propfind, &ab)
			if err != nil {
				return nil, err
			}
			resps = append(resps, resps_...)
		}
	}
	return resps, nil
}

func (b *backend) propFindCalendarObject(ctx context.Context, propfind *internal.PropFind, co *CalendarObject) (*internal.Response, error) {
	props := map[xml.Name]internal.PropFindFunc{
		internal.CurrentUserPrincipalName: func(*internal.RawXMLValue) (interface{}, error) {
			path, err := b.Backend.CurrentUserPrincipal(ctx)
			if err != nil {
				return nil, err
			}
			return &internal.CurrentUserPrincipal{Href: internal.Href{Path: path}}, nil
		},
		internal.GetContentTypeName: func(*internal.RawXMLValue) (interface{}, error) {
			return &internal.GetContentType{Type: ical.MIMEType}, nil
		},
		// TODO: calendar-data can only be used in REPORT requests
		calendarDataName: func(*internal.RawXMLValue) (interface{}, error) {
			var buf bytes.Buffer
			if err := ical.NewEncoder(&buf).Encode(co.Data); err != nil {
				return nil, err
			}

			return &calendarDataResp{Data: buf.Bytes()}, nil
		},
	}

	if co.ContentLength > 0 {
		props[internal.GetContentLengthName] = func(*internal.RawXMLValue) (interface{}, error) {
			return &internal.GetContentLength{Length: co.ContentLength}, nil
		}
	}
	if !co.ModTime.IsZero() {
		props[internal.GetLastModifiedName] = func(*internal.RawXMLValue) (interface{}, error) {
			return &internal.GetLastModified{LastModified: internal.Time(co.ModTime)}, nil
		}
	}

	if co.ETag != "" {
		props[internal.GetETagName] = func(*internal.RawXMLValue) (interface{}, error) {
			return &internal.GetETag{ETag: internal.ETag(co.ETag)}, nil
		}
	}

	return internal.NewPropFindResponse(co.Path, propfind, props)
}

func (b *backend) propFindAllCalendarObjects(ctx context.Context, propfind *internal.PropFind, cal *Calendar) ([]internal.Response, error) {
	var dataReq CalendarCompRequest
	aos, err := b.Backend.ListCalendarObjects(ctx, cal.Path, &dataReq)
	if err != nil {
		return nil, err
	}

	var resps []internal.Response
	for _, ao := range aos {
		resp, err := b.propFindCalendarObject(ctx, propfind, &ao)
		if err != nil {
			return nil, err
		}
		resps = append(resps, *resp)
	}
	return resps, nil
}

func (b *backend) PropPatch(r *http.Request, update *internal.PropertyUpdate) (*internal.Response, error) {
	return nil, internal.HTTPErrorf(http.StatusNotImplemented, "caldav: PropPatch not implemented")
}

func (b *backend) Put(w http.ResponseWriter, r *http.Request) error {
	ifNoneMatch := webdav.ConditionalMatch(r.Header.Get("If-None-Match"))
	ifMatch := webdav.ConditionalMatch(r.Header.Get("If-Match"))

	opts := PutCalendarObjectOptions{
		IfNoneMatch: ifNoneMatch,
		IfMatch:     ifMatch,
	}

	t, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return internal.HTTPErrorf(http.StatusBadRequest, "caldav: malformed Content-Type: %v", err)
	}
	if t != ical.MIMEType {
		// TODO: send CALDAV:supported-calendar-data error
		return internal.HTTPErrorf(http.StatusBadRequest, "caldav: unsupported Content-Type %q", t)
	}

	// TODO: check CALDAV:max-resource-size precondition
	cal, err := ical.NewDecoder(r.Body).Decode()
	if err != nil {
		// TODO: send CALDAV:valid-calendar-data error
		return internal.HTTPErrorf(http.StatusBadRequest, "caldav: failed to parse iCalendar: %v", err)
	}

	co, err := b.Backend.PutCalendarObject(r.Context(), r.URL.Path, cal, &opts)
	if err != nil {
		return err
	}

	if co.ETag != "" {
		w.Header().Set("ETag", internal.ETag(co.ETag).String())
	}
	if !co.ModTime.IsZero() {
		w.Header().Set("Last-Modified", co.ModTime.UTC().Format(http.TimeFormat))
	}
	if co.Path != "" {
		w.Header().Set("Location", co.Path)
	}

	// TODO: http.StatusNoContent if the resource already existed
	w.WriteHeader(http.StatusCreated)

	return nil
}

func (b *backend) Delete(r *http.Request) error {
	return b.Backend.DeleteCalendarObject(r.Context(), r.URL.Path)
}

func (b *backend) Mkcol(r *http.Request) error {
	if b.resourceTypeAtPath(r.URL.Path) != resourceTypeCalendar {
		return internal.HTTPErrorf(http.StatusForbidden, "caldav: calendar creation not allowed at given location")
	}

	cal := Calendar{
		Path: r.URL.Path,
	}

	if !internal.IsRequestBodyEmpty(r) {
		var m mkcolReq
		if err := internal.DecodeXMLRequest(r, &m); err != nil {
			return internal.HTTPErrorf(http.StatusBadRequest, "carddav: error parsing mkcol request: %s", err.Error())
		}

		if !m.ResourceType.Is(internal.CollectionName) || !m.ResourceType.Is(calendarName) {
			return internal.HTTPErrorf(http.StatusBadRequest, "carddav: unexpected resource type")
		}
		cal.Name = m.DisplayName
		cal.Description = m.Description
		cal.Color = strings.TrimSpace(m.CalendarColor)
		cal.Timezone = strings.TrimSpace(m.CalemdarTimeZone)

		cal.SupportedComponentSet = make([]string, len(m.SupportedCalendarComponentSet.Comp))
		for i, v := range m.SupportedCalendarComponentSet.Comp {
			cal.SupportedComponentSet[i] = v.Name
		}
	}

	return b.Backend.CreateCalendar(r.Context(), &cal)
}

func (b *backend) Copy(r *http.Request, dest *internal.Href, recursive, overwrite bool) (created bool, err error) {
	return false, internal.HTTPErrorf(http.StatusNotImplemented, "caldav: Copy not implemented")
}

func (b *backend) Move(r *http.Request, dest *internal.Href, overwrite bool) (created bool, err error) {
	return false, internal.HTTPErrorf(http.StatusNotImplemented, "caldav: Move not implemented")
}

// https://datatracker.ietf.org/doc/html/rfc4791#section-5.3.2.1
type PreconditionType string

const (
	PreconditionNoUIDConflict                PreconditionType = "no-uid-conflict"
	PreconditionSupportedCalendarData        PreconditionType = "supported-calendar-data"
	PreconditionSupportedCalendarComponent   PreconditionType = "supported-calendar-component"
	PreconditionValidCalendarData            PreconditionType = "valid-calendar-data"
	PreconditionValidCalendarObjectResource  PreconditionType = "valid-calendar-object-resource"
	PreconditionCalendarCollectionLocationOk PreconditionType = "calendar-collection-location-ok"
	PreconditionMaxResourceSize              PreconditionType = "max-resource-size"
	PreconditionMinDateTime                  PreconditionType = "min-date-time"
	PreconditionMaxDateTime                  PreconditionType = "max-date-time"
	PreconditionMaxInstances                 PreconditionType = "max-instances"
	PreconditionMaxAttendeesPerInstance      PreconditionType = "max-attendees-per-instance"
)

func NewPreconditionError(err PreconditionType) error {
	name := xml.Name{Space: "urn:ietf:params:xml:ns:caldav", Local: string(err)}
	elem := internal.NewRawXMLElement(name, nil, nil)
	return &internal.HTTPError{
		Code: 409,
		Err: &internal.Error{
			Raw: []internal.RawXMLValue{*elem},
		},
	}
}
