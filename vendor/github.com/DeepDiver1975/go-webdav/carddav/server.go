package carddav

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

	"github.com/DeepDiver1975/go-webdav"
	"github.com/DeepDiver1975/go-webdav/internal"
	"github.com/emersion/go-vcard"
)

type PutAddressObjectOptions struct {
	// IfNoneMatch indicates that the client does not want to overwrite
	// an existing resource.
	IfNoneMatch webdav.ConditionalMatch
	// IfMatch provides the ETag of the resource that the client intends
	// to overwrite, can be ""
	IfMatch webdav.ConditionalMatch
}

// Backend is a CardDAV server backend.
type Backend interface {
	AddressBookHomeSetPath(ctx context.Context) (string, error)
	ListAddressBooks(ctx context.Context) ([]AddressBook, error)
	GetAddressBook(ctx context.Context, path string) (*AddressBook, error)
	CreateAddressBook(ctx context.Context, addressBook *AddressBook) error
	DeleteAddressBook(ctx context.Context, path string) error
	GetAddressObject(ctx context.Context, path string, req *AddressDataRequest) (*AddressObject, error)
	ListAddressObjects(ctx context.Context, path string, req *AddressDataRequest) ([]AddressObject, error)
	QueryAddressObjects(ctx context.Context, path string, query *AddressBookQuery) ([]AddressObject, error)
	PutAddressObject(ctx context.Context, path string, card vcard.Card, opts *PutAddressObjectOptions) (*AddressObject, error)
	DeleteAddressObject(ctx context.Context, path string) error

	webdav.UserPrincipalBackend
}

// Handler handles CardDAV HTTP requests. It can be used to create a CardDAV
// server.
type Handler struct {
	Backend Backend
	Prefix  string
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Backend == nil {
		http.Error(w, "carddav: no backend available", http.StatusInternalServerError)
		return
	}

	if r.URL.Path == "/.well-known/carddav" {
		principalPath, err := h.Backend.CurrentUserPrincipal(r.Context())
		if err != nil {
			http.Error(w, "carddav: failed to determine current user principal", http.StatusInternalServerError)
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
	return internal.HTTPErrorf(http.StatusBadRequest, "carddav: expected addressbook-query or addressbook-multiget element in REPORT request")
}

func decodePropFilter(el *propFilter) (*PropFilter, error) {
	pf := &PropFilter{Name: el.Name, Test: FilterTest(el.Test)}
	if el.IsNotDefined != nil {
		if len(el.TextMatches) > 0 || len(el.Params) > 0 {
			return nil, fmt.Errorf("carddav: failed to parse prop-filter: if is-not-defined is provided, text-match or param-filter can't be provided")
		}
		pf.IsNotDefined = true
	}
	for _, tm := range el.TextMatches {
		pf.TextMatches = append(pf.TextMatches, *decodeTextMatch(&tm))
	}
	for _, paramEl := range el.Params {
		param, err := decodeParamFilter(&paramEl)
		if err != nil {
			return nil, err
		}
		pf.Params = append(pf.Params, *param)
	}
	return pf, nil
}

func decodeParamFilter(el *paramFilter) (*ParamFilter, error) {
	pf := &ParamFilter{Name: el.Name}
	if el.IsNotDefined != nil {
		if el.TextMatch != nil {
			return nil, fmt.Errorf("carddav: failed to parse param-filter: if is-not-defined is provided, text-match can't be provided")
		}
		pf.IsNotDefined = true
	}
	if el.TextMatch != nil {
		pf.TextMatch = decodeTextMatch(el.TextMatch)
	}
	return pf, nil
}

func decodeTextMatch(tm *textMatch) *TextMatch {
	return &TextMatch{
		Text:            tm.Text,
		NegateCondition: bool(tm.NegateCondition),
		MatchType:       MatchType(tm.MatchType),
	}
}

func decodeAddressDataReq(addressData *addressDataReq) (*AddressDataRequest, error) {
	if addressData.Allprop != nil && len(addressData.Props) > 0 {
		return nil, internal.HTTPErrorf(http.StatusBadRequest, "carddav: only one of allprop or prop can be specified in address-data")
	}

	req := &AddressDataRequest{AllProp: addressData.Allprop != nil}
	for _, p := range addressData.Props {
		req.Props = append(req.Props, p.Name)
	}
	return req, nil
}

func (h *Handler) handleQuery(r *http.Request, w http.ResponseWriter, query *addressbookQuery) error {
	var q AddressBookQuery
	if query.Prop != nil {
		var addressData addressDataReq
		if err := query.Prop.Decode(&addressData); err != nil && !internal.IsNotFound(err) {
			return err
		}
		req, err := decodeAddressDataReq(&addressData)
		if err != nil {
			return err
		}
		q.DataRequest = *req
	}
	q.FilterTest = FilterTest(query.Filter.Test)
	for _, el := range query.Filter.Props {
		pf, err := decodePropFilter(&el)
		if err != nil {
			return &internal.HTTPError{Code: http.StatusBadRequest, Err: err}
		}
		q.PropFilters = append(q.PropFilters, *pf)
	}
	if query.Limit != nil {
		q.Limit = int(query.Limit.NResults)
		if q.Limit <= 0 {
			return internal.ServeMultiStatus(w, internal.NewMultiStatus())
		}
	}

	aos, err := h.Backend.QueryAddressObjects(r.Context(), r.URL.Path, &q)
	if err != nil {
		return err
	}

	var resps []internal.Response
	for _, ao := range aos {
		b := backend{
			Backend: h.Backend,
			Prefix:  strings.TrimSuffix(h.Prefix, "/"),
		}
		propfind := internal.PropFind{
			Prop:     query.Prop,
			AllProp:  query.AllProp,
			PropName: query.PropName,
		}
		resp, err := b.propFindAddressObject(r.Context(), &propfind, &ao)
		if err != nil {
			return err
		}
		resps = append(resps, *resp)
	}

	ms := internal.NewMultiStatus(resps...)
	return internal.ServeMultiStatus(w, ms)
}

func (h *Handler) handleMultiget(ctx context.Context, w http.ResponseWriter, multiget *addressbookMultiget) error {
	var dataReq AddressDataRequest
	if multiget.Prop != nil {
		var addressData addressDataReq
		if err := multiget.Prop.Decode(&addressData); err != nil && !internal.IsNotFound(err) {
			return err
		}
		decoded, err := decodeAddressDataReq(&addressData)
		if err != nil {
			return err
		}
		dataReq = *decoded
	}

	var resps []internal.Response
	for _, href := range multiget.Hrefs {
		ao, err := h.Backend.GetAddressObject(ctx, href.Path, &dataReq)
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
		resp, err := b.propFindAddressObject(ctx, &propfind, ao)
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
	resourceTypeAddressBookHomeSet
	resourceTypeAddressBook
	resourceTypeAddressObject
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
	caps = []string{"addressbook"}

	if b.resourceTypeAtPath(r.URL.Path) != resourceTypeAddressObject {
		// Note: some clients assume the address book is read-only when
		// DELETE/MKCOL are missing
		return caps, []string{http.MethodOptions, "PROPFIND", "REPORT", "DELETE", "MKCOL"}, nil
	}

	var dataReq AddressDataRequest
	_, err = b.Backend.GetAddressObject(r.Context(), r.URL.Path, &dataReq)
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
	var dataReq AddressDataRequest
	if r.Method != http.MethodHead {
		dataReq.AllProp = true
	}
	ao, err := b.Backend.GetAddressObject(r.Context(), r.URL.Path, &dataReq)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", vcard.MIMEType)
	if ao.ContentLength > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(ao.ContentLength, 10))
	}
	if ao.ETag != "" {
		w.Header().Set("ETag", internal.ETag(ao.ETag).String())
	}
	if !ao.ModTime.IsZero() {
		w.Header().Set("Last-Modified", ao.ModTime.UTC().Format(http.TimeFormat))
	}

	if r.Method != http.MethodHead {
		return vcard.NewEncoder(w).Encode(ao.Card)
	}
	return nil
}

func (b *backend) PropFind(r *http.Request, propfind *internal.PropFind, depth internal.Depth) (*internal.MultiStatus, error) {
	resType := b.resourceTypeAtPath(r.URL.Path)

	var dataReq AddressDataRequest
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
					resps_, err := b.propFindAllAddressBooks(r.Context(), propfind, true)
					if err != nil {
						return nil, err
					}
					resps = append(resps, resps_...)
				}
			}
		}
	case resourceTypeAddressBookHomeSet:
		homeSetPath, err := b.Backend.AddressBookHomeSetPath(r.Context())
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
				resps_, err := b.propFindAllAddressBooks(r.Context(), propfind, recurse)
				if err != nil {
					return nil, err
				}
				resps = append(resps, resps_...)
			}
		}
	case resourceTypeAddressBook:
		ab, err := b.Backend.GetAddressBook(r.Context(), r.URL.Path)
		if err != nil {
			return nil, err
		}
		resp, err := b.propFindAddressBook(r.Context(), propfind, ab)
		if err != nil {
			return nil, err
		}
		resps = append(resps, *resp)
		if depth != internal.DepthZero {
			resps_, err := b.propFindAllAddressObjects(r.Context(), propfind, ab)
			if err != nil {
				return nil, err
			}
			resps = append(resps, resps_...)
		}
	case resourceTypeAddressObject:
		ao, err := b.Backend.GetAddressObject(r.Context(), r.URL.Path, &dataReq)
		if err != nil {
			return nil, err
		}

		resp, err := b.propFindAddressObject(r.Context(), propfind, ao)
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
	homeSetPath, err := b.Backend.AddressBookHomeSetPath(ctx)
	if err != nil {
		return nil, err
	}

	props := map[xml.Name]internal.PropFindFunc{
		internal.CurrentUserPrincipalName: func(*internal.RawXMLValue) (interface{}, error) {
			return &internal.CurrentUserPrincipal{Href: internal.Href{Path: principalPath}}, nil
		},
		addressBookHomeSetName: func(*internal.RawXMLValue) (interface{}, error) {
			return &addressbookHomeSet{Href: internal.Href{Path: homeSetPath}}, nil
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
	homeSetPath, err := b.Backend.AddressBookHomeSetPath(ctx)
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

func (b *backend) propFindAddressBook(ctx context.Context, propfind *internal.PropFind, ab *AddressBook) (*internal.Response, error) {
	props := map[xml.Name]internal.PropFindFunc{
		internal.CurrentUserPrincipalName: func(*internal.RawXMLValue) (interface{}, error) {
			path, err := b.Backend.CurrentUserPrincipal(ctx)
			if err != nil {
				return nil, err
			}
			return &internal.CurrentUserPrincipal{Href: internal.Href{Path: path}}, nil
		},
		internal.ResourceTypeName: func(*internal.RawXMLValue) (interface{}, error) {
			return internal.NewResourceType(internal.CollectionName, addressBookName), nil
		},
		supportedAddressDataName: func(*internal.RawXMLValue) (interface{}, error) {
			return &supportedAddressData{
				Types: []addressDataType{
					{ContentType: vcard.MIMEType, Version: "3.0"},
					{ContentType: vcard.MIMEType, Version: "4.0"},
				},
			}, nil
		},
	}

	if ab.Name != "" {
		props[internal.DisplayNameName] = func(*internal.RawXMLValue) (interface{}, error) {
			return &internal.DisplayName{Name: ab.Name}, nil
		}
	}
	if ab.Description != "" {
		props[addressBookDescriptionName] = func(*internal.RawXMLValue) (interface{}, error) {
			return &addressbookDescription{Description: ab.Description}, nil
		}
	}
	if ab.MaxResourceSize > 0 {
		props[maxResourceSizeName] = func(*internal.RawXMLValue) (interface{}, error) {
			return &maxResourceSize{Size: ab.MaxResourceSize}, nil
		}
	}

	return internal.NewPropFindResponse(ab.Path, propfind, props)
}

func (b *backend) propFindAllAddressBooks(ctx context.Context, propfind *internal.PropFind, recurse bool) ([]internal.Response, error) {
	abs, err := b.Backend.ListAddressBooks(ctx)
	if err != nil {
		return nil, err
	}

	var resps []internal.Response
	for _, ab := range abs {
		resp, err := b.propFindAddressBook(ctx, propfind, &ab)
		if err != nil {
			return nil, err
		}
		resps = append(resps, *resp)
		if recurse {
			resps_, err := b.propFindAllAddressObjects(ctx, propfind, &ab)
			if err != nil {
				return nil, err
			}
			resps = append(resps, resps_...)
		}
	}
	return resps, nil
}

func (b *backend) propFindAddressObject(ctx context.Context, propfind *internal.PropFind, ao *AddressObject) (*internal.Response, error) {
	props := map[xml.Name]internal.PropFindFunc{
		internal.CurrentUserPrincipalName: func(*internal.RawXMLValue) (interface{}, error) {
			path, err := b.Backend.CurrentUserPrincipal(ctx)
			if err != nil {
				return nil, err
			}
			return &internal.CurrentUserPrincipal{Href: internal.Href{Path: path}}, nil
		},
		internal.GetContentTypeName: func(*internal.RawXMLValue) (interface{}, error) {
			return &internal.GetContentType{Type: vcard.MIMEType}, nil
		},
		// TODO: address-data can only be used in REPORT requests
		addressDataName: func(*internal.RawXMLValue) (interface{}, error) {
			var buf bytes.Buffer
			if err := vcard.NewEncoder(&buf).Encode(ao.Card); err != nil {
				return nil, err
			}

			return &addressDataResp{Data: buf.Bytes()}, nil
		},
	}

	if ao.ContentLength > 0 {
		props[internal.GetContentLengthName] = func(*internal.RawXMLValue) (interface{}, error) {
			return &internal.GetContentLength{Length: ao.ContentLength}, nil
		}
	}
	if !ao.ModTime.IsZero() {
		props[internal.GetLastModifiedName] = func(*internal.RawXMLValue) (interface{}, error) {
			return &internal.GetLastModified{LastModified: internal.Time(ao.ModTime)}, nil
		}
	}

	if ao.ETag != "" {
		props[internal.GetETagName] = func(*internal.RawXMLValue) (interface{}, error) {
			return &internal.GetETag{ETag: internal.ETag(ao.ETag)}, nil
		}
	}

	return internal.NewPropFindResponse(ao.Path, propfind, props)
}

func (b *backend) propFindAllAddressObjects(ctx context.Context, propfind *internal.PropFind, ab *AddressBook) ([]internal.Response, error) {
	var dataReq AddressDataRequest
	aos, err := b.Backend.ListAddressObjects(ctx, ab.Path, &dataReq)
	if err != nil {
		return nil, err
	}

	var resps []internal.Response
	for _, ao := range aos {
		resp, err := b.propFindAddressObject(ctx, propfind, &ao)
		if err != nil {
			return nil, err
		}
		resps = append(resps, *resp)
	}
	return resps, nil
}

func (b *backend) PropPatch(r *http.Request, update *internal.PropertyUpdate) (*internal.Response, error) {
	homeSetPath, err := b.Backend.AddressBookHomeSetPath(r.Context())
	if err != nil {
		return nil, err
	}

	resp := internal.NewOKResponse(r.URL.Path)

	if r.URL.Path == homeSetPath {
		// TODO: support PROPPATCH for address books
		for _, prop := range update.Remove {
			emptyVal := internal.NewRawXMLElement(prop.Prop.XMLName, nil, nil)
			if err := resp.EncodeProp(http.StatusNotImplemented, emptyVal); err != nil {
				return nil, err
			}
		}
		for _, prop := range update.Set {
			emptyVal := internal.NewRawXMLElement(prop.Prop.XMLName, nil, nil)
			if err := resp.EncodeProp(http.StatusNotImplemented, emptyVal); err != nil {
				return nil, err
			}
		}
	} else {
		for _, prop := range update.Remove {
			emptyVal := internal.NewRawXMLElement(prop.Prop.XMLName, nil, nil)
			if err := resp.EncodeProp(http.StatusMethodNotAllowed, emptyVal); err != nil {
				return nil, err
			}
		}
		for _, prop := range update.Set {
			emptyVal := internal.NewRawXMLElement(prop.Prop.XMLName, nil, nil)
			if err := resp.EncodeProp(http.StatusMethodNotAllowed, emptyVal); err != nil {
				return nil, err
			}
		}
	}

	return resp, nil
}

func (b *backend) Put(w http.ResponseWriter, r *http.Request) error {
	ifNoneMatch := webdav.ConditionalMatch(r.Header.Get("If-None-Match"))
	ifMatch := webdav.ConditionalMatch(r.Header.Get("If-Match"))

	opts := PutAddressObjectOptions{
		IfNoneMatch: ifNoneMatch,
		IfMatch:     ifMatch,
	}

	t, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return internal.HTTPErrorf(http.StatusBadRequest, "carddav: malformed Content-Type: %v", err)
	}
	if t != vcard.MIMEType {
		// TODO: send CARDDAV:supported-address-data error
		return internal.HTTPErrorf(http.StatusBadRequest, "carddav: unsupporetd Content-Type %q", t)
	}

	// TODO: check CARDDAV:max-resource-size precondition
	card, err := vcard.NewDecoder(r.Body).Decode()
	if err != nil {
		// TODO: send CARDDAV:valid-address-data error
		return internal.HTTPErrorf(http.StatusBadRequest, "carddav: failed to parse vCard: %v", err)
	}

	// TODO: add support for the CARDDAV:no-uid-conflict error
	ao, err := b.Backend.PutAddressObject(r.Context(), r.URL.Path, card, &opts)
	if err != nil {
		return err
	}
	if ao.ETag != "" {
		w.Header().Set("ETag", internal.ETag(ao.ETag).String())
	}
	if !ao.ModTime.IsZero() {
		w.Header().Set("Last-Modified", ao.ModTime.UTC().Format(http.TimeFormat))
	}
	if ao.Path != "" {
		w.Header().Set("Location", ao.Path)
	}

	// TODO: http.StatusNoContent if the resource already existed
	w.WriteHeader(http.StatusCreated)

	return nil
}

func (b *backend) Delete(r *http.Request) error {
	switch b.resourceTypeAtPath(r.URL.Path) {
	case resourceTypeAddressBook:
		return b.Backend.DeleteAddressBook(r.Context(), r.URL.Path)
	case resourceTypeAddressObject:
		return b.Backend.DeleteAddressObject(r.Context(), r.URL.Path)
	}
	return internal.HTTPErrorf(http.StatusForbidden, "carddav: cannot delete resource at given location")
}

func (b *backend) Mkcol(r *http.Request) error {
	if b.resourceTypeAtPath(r.URL.Path) != resourceTypeAddressBook {
		return internal.HTTPErrorf(http.StatusForbidden, "carddav: address book creation not allowed at given location")
	}

	ab := AddressBook{
		Path: r.URL.Path,
	}

	if !internal.IsRequestBodyEmpty(r) {
		var m mkcolReq
		if err := internal.DecodeXMLRequest(r, &m); err != nil {
			return internal.HTTPErrorf(http.StatusBadRequest, "carddav: error parsing mkcol request: %s", err.Error())
		}

		if !m.ResourceType.Is(internal.CollectionName) || !m.ResourceType.Is(addressBookName) {
			return internal.HTTPErrorf(http.StatusBadRequest, "carddav: unexpected resource type")
		}
		ab.Name = m.DisplayName
		ab.Description = m.Description.Description
		// TODO ...
	}

	return b.Backend.CreateAddressBook(r.Context(), &ab)
}

func (b *backend) Copy(r *http.Request, dest *internal.Href, recursive, overwrite bool) (created bool, err error) {
	return false, internal.HTTPErrorf(http.StatusNotImplemented, "carddav: Copy not implemented")
}

func (b *backend) Move(r *http.Request, dest *internal.Href, overwrite bool) (created bool, err error) {
	return false, internal.HTTPErrorf(http.StatusNotImplemented, "carddav: Move not implemented")
}

// PreconditionType as defined in https://tools.ietf.org/rfcmarkup?doc=6352#section-6.3.2.1
type PreconditionType string

const (
	PreconditionNoUIDConflict        PreconditionType = "no-uid-conflict"
	PreconditionSupportedAddressData PreconditionType = "supported-address-data"
	PreconditionValidAddressData     PreconditionType = "valid-address-data"
	PreconditionMaxResourceSize      PreconditionType = "max-resource-size"
)

func NewPreconditionError(err PreconditionType) error {
	name := xml.Name{Space: "urn:ietf:params:xml:ns:carddav", Local: string(err)}
	elem := internal.NewRawXMLElement(name, nil, nil)
	return &internal.HTTPError{
		Code: 409,
		Err: &internal.Error{
			Raw: []internal.RawXMLValue{*elem},
		},
	}
}
