package carddav

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/DeepDiver1975/go-webdav"
	"github.com/DeepDiver1975/go-webdav/internal"
	"github.com/emersion/go-vcard"
)

// DiscoverContextURL performs a DNS-based CardDAV service discovery as
// described in RFC 6352 section 11. It returns the URL to the CardDAV server.
func DiscoverContextURL(ctx context.Context, domain string) (string, error) {
	return internal.DiscoverContextURL(ctx, "carddavs", domain)
}

// Client provides access to a remote CardDAV server.
type Client struct {
	*webdav.Client

	ic *internal.Client
}

func NewClient(c webdav.HTTPClient, endpoint string) (*Client, error) {
	wc, err := webdav.NewClient(c, endpoint)
	if err != nil {
		return nil, err
	}
	ic, err := internal.NewClient(c, endpoint)
	if err != nil {
		return nil, err
	}
	return &Client{wc, ic}, nil
}

func (c *Client) HasSupport(ctx context.Context) error {
	classes, _, err := c.ic.Options(ctx, "")
	if err != nil {
		return err
	}

	if !classes["addressbook"] {
		return fmt.Errorf("carddav: server doesn't support the DAV addressbook class")
	}
	return nil
}

func (c *Client) FindAddressBookHomeSet(ctx context.Context, principal string) (string, error) {
	propfind := internal.NewPropNamePropFind(addressBookHomeSetName)
	resp, err := c.ic.PropFindFlat(ctx, principal, propfind)
	if err != nil {
		return "", err
	}

	var prop addressbookHomeSet
	if err := resp.DecodeProp(&prop); err != nil {
		return "", err
	}

	return prop.Href.Path, nil
}

func decodeSupportedAddressData(supported *supportedAddressData) []AddressDataType {
	l := make([]AddressDataType, len(supported.Types))
	for i, t := range supported.Types {
		l[i] = AddressDataType{t.ContentType, t.Version}
	}
	return l
}

func (c *Client) FindAddressBooks(ctx context.Context, addressBookHomeSet string) ([]AddressBook, error) {
	propfind := internal.NewPropNamePropFind(
		internal.ResourceTypeName,
		internal.DisplayNameName,
		addressBookDescriptionName,
		maxResourceSizeName,
		supportedAddressDataName,
	)
	ms, err := c.ic.PropFind(ctx, addressBookHomeSet, internal.DepthOne, propfind)
	if err != nil {
		return nil, err
	}

	l := make([]AddressBook, 0, len(ms.Responses))
	for _, resp := range ms.Responses {
		path, err := resp.Path()
		if err != nil {
			return nil, err
		}

		var resType internal.ResourceType
		if err := resp.DecodeProp(&resType); err != nil {
			return nil, err
		}
		if !resType.Is(addressBookName) {
			continue
		}

		var desc addressbookDescription
		if err := resp.DecodeProp(&desc); err != nil && !internal.IsNotFound(err) {
			return nil, err
		}

		var dispName internal.DisplayName
		if err := resp.DecodeProp(&dispName); err != nil && !internal.IsNotFound(err) {
			return nil, err
		}

		var maxResSize maxResourceSize
		if err := resp.DecodeProp(&maxResSize); err != nil && !internal.IsNotFound(err) {
			return nil, err
		}
		if maxResSize.Size < 0 {
			return nil, fmt.Errorf("carddav: max-resource-size must be a positive integer")
		}

		var supported supportedAddressData
		if err := resp.DecodeProp(&supported); err != nil && !internal.IsNotFound(err) {
			return nil, err
		}

		l = append(l, AddressBook{
			Path:                 path,
			Name:                 dispName.Name,
			Description:          desc.Description,
			MaxResourceSize:      maxResSize.Size,
			SupportedAddressData: decodeSupportedAddressData(&supported),
		})
	}

	return l, nil
}

func encodeAddressPropReq(req *AddressDataRequest) (*internal.Prop, error) {
	var addrDataReq addressDataReq
	if req.AllProp {
		addrDataReq.Allprop = &struct{}{}
	} else {
		for _, name := range req.Props {
			addrDataReq.Props = append(addrDataReq.Props, prop{Name: name})
		}
	}

	getLastModReq := internal.NewRawXMLElement(internal.GetLastModifiedName, nil, nil)
	getETagReq := internal.NewRawXMLElement(internal.GetETagName, nil, nil)
	return internal.EncodeProp(&addrDataReq, getLastModReq, getETagReq)
}

func encodePropFilter(pf *PropFilter) (*propFilter, error) {
	el := &propFilter{Name: pf.Name, Test: filterTest(pf.Test)}
	if pf.IsNotDefined {
		if len(pf.TextMatches) > 0 || len(pf.Params) > 0 {
			return nil, fmt.Errorf("carddav: failed to encode PropFilter: IsNotDefined cannot be set with TextMatches or Params")
		}
		el.IsNotDefined = &struct{}{}
	}
	for _, tm := range pf.TextMatches {
		el.TextMatches = append(el.TextMatches, *encodeTextMatch(&tm))
	}
	for _, param := range pf.Params {
		paramEl, err := encodeParamFilter(&param)
		if err != nil {
			return nil, err
		}
		el.Params = append(el.Params, *paramEl)
	}
	return el, nil
}

func encodeParamFilter(pf *ParamFilter) (*paramFilter, error) {
	el := &paramFilter{Name: pf.Name}
	if pf.IsNotDefined {
		if pf.TextMatch != nil {
			return nil, fmt.Errorf("carddav: failed to encode ParamFilter: only one of IsNotDefined or TextMatch can be set")
		}
		el.IsNotDefined = &struct{}{}
	}
	if pf.TextMatch != nil {
		el.TextMatch = encodeTextMatch(pf.TextMatch)
	}
	return el, nil
}

func encodeTextMatch(tm *TextMatch) *textMatch {
	return &textMatch{
		Text:            tm.Text,
		NegateCondition: negateCondition(tm.NegateCondition),
		MatchType:       matchType(tm.MatchType),
	}
}

func decodeAddressList(ms *internal.MultiStatus) ([]AddressObject, error) {
	addrs := make([]AddressObject, 0, len(ms.Responses))
	for _, resp := range ms.Responses {
		path, err := resp.Path()
		if err != nil {
			return nil, err
		}

		var addrData addressDataResp
		if err := resp.DecodeProp(&addrData); err != nil {
			return nil, err
		}

		var getLastMod internal.GetLastModified
		if err := resp.DecodeProp(&getLastMod); err != nil && !internal.IsNotFound(err) {
			return nil, err
		}

		var getETag internal.GetETag
		if err := resp.DecodeProp(&getETag); err != nil && !internal.IsNotFound(err) {
			return nil, err
		}

		var getContentLength internal.GetContentLength
		if err := resp.DecodeProp(&getContentLength); err != nil && !internal.IsNotFound(err) {
			return nil, err
		}

		r := bytes.NewReader(addrData.Data)
		card, err := vcard.NewDecoder(r).Decode()
		if err != nil {
			return nil, err
		}

		addrs = append(addrs, AddressObject{
			Path:          path,
			ModTime:       time.Time(getLastMod.LastModified),
			ContentLength: getContentLength.Length,
			ETag:          string(getETag.ETag),
			Card:          card,
		})
	}

	return addrs, nil
}

func (c *Client) QueryAddressBook(ctx context.Context, addressBook string, query *AddressBookQuery) ([]AddressObject, error) {
	propReq, err := encodeAddressPropReq(&query.DataRequest)
	if err != nil {
		return nil, err
	}

	addressbookQuery := addressbookQuery{Prop: propReq}
	addressbookQuery.Filter.Test = filterTest(query.FilterTest)
	for _, pf := range query.PropFilters {
		el, err := encodePropFilter(&pf)
		if err != nil {
			return nil, err
		}
		addressbookQuery.Filter.Props = append(addressbookQuery.Filter.Props, *el)
	}
	if query.Limit > 0 {
		addressbookQuery.Limit = &limit{NResults: uint(query.Limit)}
	}

	req, err := c.ic.NewXMLRequest("REPORT", addressBook, &addressbookQuery)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Depth", "1")

	ms, err := c.ic.DoMultiStatus(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	return decodeAddressList(ms)
}

func (c *Client) MultiGetAddressBook(ctx context.Context, path string, multiGet *AddressBookMultiGet) ([]AddressObject, error) {
	propReq, err := encodeAddressPropReq(&multiGet.DataRequest)
	if err != nil {
		return nil, err
	}

	addressbookMultiget := addressbookMultiget{Prop: propReq}

	if len(multiGet.Paths) == 0 {
		href := internal.Href{Path: path}
		addressbookMultiget.Hrefs = []internal.Href{href}
	} else {
		addressbookMultiget.Hrefs = make([]internal.Href, len(multiGet.Paths))
		for i, p := range multiGet.Paths {
			addressbookMultiget.Hrefs[i] = internal.Href{Path: p}
		}
	}

	req, err := c.ic.NewXMLRequest("REPORT", path, &addressbookMultiget)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Depth", "1")

	ms, err := c.ic.DoMultiStatus(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	return decodeAddressList(ms)
}

func populateAddressObject(ao *AddressObject, h http.Header) error {
	if loc := h.Get("Location"); loc != "" {
		u, err := url.Parse(loc)
		if err != nil {
			return err
		}
		ao.Path = u.Path
	}
	if etag := h.Get("ETag"); etag != "" {
		etag, err := strconv.Unquote(etag)
		if err != nil {
			return err
		}
		ao.ETag = etag
	}
	if contentLength := h.Get("Content-Length"); contentLength != "" {
		n, err := strconv.ParseInt(contentLength, 10, 64)
		if err != nil {
			return err
		}
		ao.ContentLength = n
	}
	if lastModified := h.Get("Last-Modified"); lastModified != "" {
		t, err := http.ParseTime(lastModified)
		if err != nil {
			return err
		}
		ao.ModTime = t
	}

	return nil
}

func (c *Client) GetAddressObject(ctx context.Context, path string) (*AddressObject, error) {
	req, err := c.ic.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", vcard.MIMEType)

	resp, err := c.ic.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(mediaType, vcard.MIMEType) {
		return nil, fmt.Errorf("carddav: expected Content-Type %q, got %q", vcard.MIMEType, mediaType)
	}

	card, err := vcard.NewDecoder(resp.Body).Decode()
	if err != nil {
		return nil, err
	}

	ao := &AddressObject{
		Path: resp.Request.URL.Path,
		Card: card,
	}
	if err := populateAddressObject(ao, resp.Header); err != nil {
		return nil, err
	}
	return ao, nil
}

func (c *Client) PutAddressObject(ctx context.Context, path string, card vcard.Card) (*AddressObject, error) {
	// TODO: add support for If-None-Match and If-Match

	// TODO: some servers want a Content-Length header, so we can't stream the
	// request body here. See the Radicale issue:
	// https://github.com/Kozea/Radicale/issues/1016

	//pr, pw := io.Pipe()
	//go func() {
	//	err := vcard.NewEncoder(pw).Encode(card)
	//	pw.CloseWithError(err)
	//}()

	var buf bytes.Buffer
	if err := vcard.NewEncoder(&buf).Encode(card); err != nil {
		return nil, err
	}

	req, err := c.ic.NewRequest(http.MethodPut, path, &buf)
	if err != nil {
		//pr.Close()
		return nil, err
	}
	req.Header.Set("Content-Type", vcard.MIMEType)

	resp, err := c.ic.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	ao := &AddressObject{Path: path}
	if err := populateAddressObject(ao, resp.Header); err != nil {
		return nil, err
	}
	return ao, nil
}

// SyncCollection performs a collection synchronization operation on the
// specified resource, as defined in RFC 6578.
func (c *Client) SyncCollection(ctx context.Context, path string, query *SyncQuery) (*SyncResponse, error) {
	var limit *internal.Limit
	if query.Limit > 0 {
		limit = &internal.Limit{NResults: uint(query.Limit)}
	}

	propReq, err := encodeAddressPropReq(&query.DataRequest)
	if err != nil {
		return nil, err
	}

	ms, err := c.ic.SyncCollection(ctx, path, query.SyncToken, internal.DepthOne, limit, propReq)
	if err != nil {
		return nil, err
	}

	ret := &SyncResponse{SyncToken: ms.SyncToken}
	for _, resp := range ms.Responses {
		p, err := resp.Path()
		if err != nil {
			if err, ok := err.(*internal.HTTPError); ok && err.Code == http.StatusNotFound {
				ret.Deleted = append(ret.Deleted, p)
				continue
			}
			return nil, err
		}

		if p == path || path == fmt.Sprintf("%s/", p) {
			continue
		}

		var getLastMod internal.GetLastModified
		if err := resp.DecodeProp(&getLastMod); err != nil && !internal.IsNotFound(err) {
			return nil, err
		}

		var getETag internal.GetETag
		if err := resp.DecodeProp(&getETag); err != nil && !internal.IsNotFound(err) {
			return nil, err
		}

		o := AddressObject{
			Path:    p,
			ModTime: time.Time(getLastMod.LastModified),
			ETag:    string(getETag.ETag),
		}
		ret.Updated = append(ret.Updated, o)
	}

	return ret, nil
}
