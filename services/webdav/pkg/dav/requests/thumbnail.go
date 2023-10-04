package requests

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"

	providerv1beta1 "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/cs3org/reva/v2/pkg/storagespace"
	"github.com/cs3org/reva/v2/pkg/utils"
	"github.com/go-chi/chi/v5"

	"github.com/owncloud/ocis/v2/services/webdav/pkg/constants"
)

const (
	// DefaultWidth defines the default width of a thumbnail
	DefaultWidth = 32
	// DefaultHeight defines the default height of a thumbnail
	DefaultHeight = 32
)

// ThumbnailRequest combines all parameters provided when requesting a thumbnail
type ThumbnailRequest struct {
	// The file path of the source file
	Filepath string
	// The file name of the source file including the extension
	Filename string
	// The file extension
	Extension string
	// The requested width of the thumbnail
	Width int32
	// The requested height of the thumbnail
	Height int32
	// In case of a public share the public link token.
	PublicLinkToken string
	// Indicates which image processor to use
	Processor string
	// The Identifier from the requested URL
	Identifier string
}

func addMissingStorageID(id string) string {
	rid := providerv1beta1.ResourceId{}
	rid.StorageId, rid.SpaceId, rid.OpaqueId, _ = storagespace.SplitID(id)
	if rid.StorageId == "" && rid.SpaceId == utils.ShareStorageSpaceID {
		rid.StorageId = utils.ShareStorageProviderID
	}
	return storagespace.FormatResourceID(rid)
}

// ParseThumbnailRequest extracts all required parameters from a http request.
func ParseThumbnailRequest(r *http.Request) (*ThumbnailRequest, error) {
	ctx := r.Context()

	fp := ctx.Value(constants.ContextKeyPath).(string)

	id := ""
	v := ctx.Value(constants.ContextKeyID)
	if v != nil {
		id = addMissingStorageID(v.(string))
	}

	q := r.URL.Query()
	width, height, err := parseDimensions(q)
	if err != nil {
		return nil, err
	}

	return &ThumbnailRequest{
		Filepath:        fp,
		Filename:        filepath.Base(fp),
		Extension:       filepath.Ext(fp),
		Width:           int32(width),
		Height:          int32(height),
		Processor:       q.Get("processor"),
		PublicLinkToken: chi.URLParam(r, "token"),
		Identifier:      id,
	}, nil
}

func parseDimensions(q url.Values) (int64, int64, error) {
	width, err := parseDimension(q.Get("x"), "width", DefaultWidth)
	if err != nil {
		return 0, 0, err
	}
	height, err := parseDimension(q.Get("y"), "height", DefaultHeight)
	if err != nil {
		return 0, 0, err
	}
	return width, height, nil
}

func parseDimension(d, name string, defaultValue int64) (int64, error) {
	if d == "" {
		return defaultValue, nil
	}
	result, err := strconv.ParseInt(d, 10, 32)
	if err != nil || result < 1 {
		// The error message doesn't fit but for OC10 API compatibility reasons we have to set this.
		return 0, fmt.Errorf("Cannot set %s of 0 or smaller!", name)
	}
	return result, nil
}
