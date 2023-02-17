package svc

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/CiscoM31/godata"
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	revactx "github.com/cs3org/reva/v2/pkg/ctx"
	"github.com/cs3org/reva/v2/pkg/storagespace"
	"github.com/cs3org/reva/v2/pkg/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jellydator/ttlcache/v3"
	libregraph "github.com/owncloud/libre-graph-api-go"
	"github.com/owncloud/ocis/v2/ocis-pkg/service/grpc"
	v0 "github.com/owncloud/ocis/v2/protogen/gen/ocis/messages/settings/v0"
	settingssvc "github.com/owncloud/ocis/v2/protogen/gen/ocis/services/settings/v0"
	"github.com/owncloud/ocis/v2/services/graph/pkg/service/v0/errorcode"
	settingsServiceExt "github.com/owncloud/ocis/v2/services/settings/pkg/service/v0"
	"github.com/pkg/errors"
	merrors "go-micro.dev/v4/errors"
)

const (
	_spaceTypePersonal = "personal"
	_spaceTypeProject  = "project"

	_sortDescending = "desc"
)

var (
	_invalidSpaceNameCharacters = []string{`/`, `\`, `.`, `:`, `?`, `*`, `"`, `>`, `<`, `|`}
	_maxSpaceNameLength         = 255

	// ErrNameTooLong is thrown when the spacename is too long
	ErrNameTooLong = fmt.Errorf("spacename must be smaller than %d", _maxSpaceNameLength)

	// ErrNameEmpty is thrown when the spacename is empty
	ErrNameEmpty = errors.New("spacename must not be empty")

	// ErrForbiddenCharacter is thrown when the spacename contains an invalid character
	ErrForbiddenCharacter = fmt.Errorf("spacenames must not contain %v", _invalidSpaceNameCharacters)
)

// GetDrives lists all drives the current user has access to
func (g Graph) GetDrives(w http.ResponseWriter, r *http.Request) {
	g.getDrives(w, r, false)
}

// GetAllDrives lists all drives, including other user's drives, if the current
// user has the permission.
func (g Graph) GetAllDrives(w http.ResponseWriter, r *http.Request) {
	g.getDrives(w, r, true)
}

// getDrives implements the Service interface.
func (g Graph) getDrives(w http.ResponseWriter, r *http.Request, unrestricted bool) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Info().
		Interface("query", r.URL.Query()).
		Bool("unrestricted", unrestricted).
		Msg("calling get drives")
	sanitizedPath := strings.TrimPrefix(r.URL.Path, "/graph/v1.0/")
	// Parse the request with odata parser
	odataReq, err := godata.ParseRequest(r.Context(), sanitizedPath, r.URL.Query())
	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("could not get drives: query error")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}
	ctx := r.Context()

	filters, err := generateCs3Filters(odataReq)
	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("could not get drives: error parsing filters")
		errorcode.NotSupported.Render(w, r, http.StatusNotImplemented, err.Error())
		return
	}

	logger.Debug().
		Interface("filters", filters).
		Bool("unrestricted", unrestricted).
		Msg("calling list storage spaces on backend")
	res, err := g.ListStorageSpacesWithFilters(ctx, filters, unrestricted)
	switch {
	case err != nil:
		logger.Error().Err(err).Msg("could not get drives: transport error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	case res.Status.Code != cs3rpc.Code_CODE_OK:
		if res.Status.Code == cs3rpc.Code_CODE_NOT_FOUND {
			// return an empty list
			render.Status(r, http.StatusOK)
			render.JSON(w, r, &ListResponse{})
			return
		}
		logger.Debug().Str("message", res.GetStatus().GetMessage()).Msg("could not get drives: grpc error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, res.Status.Message)
		return
	}

	webDavBaseURL, err := g.getWebDavBaseURL()
	if err != nil {
		logger.Error().Err(err).Str("url", webDavBaseURL.String()).Msg("could not get drives: error parsing url")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	spaces, err := g.formatDrives(ctx, webDavBaseURL, res.StorageSpaces)
	if err != nil {
		logger.Debug().Err(err).Msg("could not get drives: error parsing grpc response")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	spaces, err = sortSpaces(odataReq, spaces)
	if err != nil {
		logger.Debug().Err(err).Msg("could not get drives: error sorting the spaces list according to query")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &ListResponse{Value: spaces})
}

// GetSingleDrive does a lookup of a single space by spaceId
func (g Graph) GetSingleDrive(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Info().Interface("query", r.URL.Query()).Msg("calling get drive")
	driveID, err := url.PathUnescape(chi.URLParam(r, "driveID"))

	if err != nil {
		logger.Debug().Err(err).Str("driveID", chi.URLParam(r, "driveID")).Msg("could not get drive: unescaping drive id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping drive id failed")
		return
	}

	if driveID == "" {
		logger.Debug().Msg("could not get drive: missing drive id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing drive id")
		return
	}

	logger.Debug().Str("driveID", driveID).Msg("calling list storage spaces with id filter")
	ctx := r.Context()

	filters := []*storageprovider.ListStorageSpacesRequest_Filter{listStorageSpacesIDFilter(driveID)}
	res, err := g.ListStorageSpacesWithFilters(ctx, filters, true)
	switch {
	case err != nil:
		logger.Error().Err(err).Msg("could not get drive: transport error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	case res.Status.Code != cs3rpc.Code_CODE_OK:
		if res.Status.Code == cs3rpc.Code_CODE_NOT_FOUND {
			// the client is doing a lookup for a specific space, therefore we need to return
			// not found to the caller
			logger.Debug().Str("driveID", driveID).Msg("could not get drive: not found")
			errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "drive not found")
			return
		}
		logger.Debug().
			Str("id", driveID).
			Str("grpcmessage", res.GetStatus().GetMessage()).
			Msg("could not get drive: grpc error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, res.Status.Message)
		return
	}

	webDavBaseURL, err := g.getWebDavBaseURL()
	if err != nil {
		logger.Error().Err(err).Str("url", webDavBaseURL.String()).Msg("could not get drive: error parsing webdav base url")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	spaces, err := g.formatDrives(ctx, webDavBaseURL, res.StorageSpaces)
	if err != nil {
		logger.Debug().Err(err).Msg("could not get drive: error parsing grpc response")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	switch num := len(spaces); {
	case num == 0:
		logger.Debug().Str("id", driveID).Msg("could not get drive: no drive returned from storage")
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "no drive returned from storage")
		return
	case num == 1:
		render.Status(r, http.StatusOK)
		render.JSON(w, r, spaces[0])
	default:
		logger.Debug().Int("number", num).Msg("could not get drive: expected to find a single drive but fetched more")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not get drive: expected to find a single drive but fetched more")
		return
	}
}

func (g Graph) canCreateSpace(ctx context.Context, ownPersonalHome bool) bool {
	pr, err := g.permissionsService.GetPermissionByID(ctx, &settingssvc.GetPermissionByIDRequest{
		PermissionId: settingsServiceExt.CreateSpacePermissionID,
	})
	if err != nil || pr.Permission == nil {
		return false
	}
	// TODO @C0rby shouldn't the permissions service check this? aka shouldn't we call CheckPermission?
	if pr.Permission.Constraint == v0.Permission_CONSTRAINT_OWN && !ownPersonalHome {
		return false
	}
	return true
}

// CreateDrive creates a storage drive (space).
func (g Graph) CreateDrive(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Info().Msg("calling create drive")
	us, ok := revactx.ContextGetUser(r.Context())
	if !ok {
		logger.Debug().Msg("could not create drive: invalid user")
		errorcode.NotAllowed.Render(w, r, http.StatusUnauthorized, "invalid user")
		return
	}

	// TODO determine if the user tries to create his own personal space and pass that as a boolean
	canCreateSpace := g.canCreateSpace(r.Context(), false)
	if !canCreateSpace {
		logger.Debug().Bool("cancreatespace", canCreateSpace).Msg("could not create drive: insufficient permissions")
		// if the permission is not existing for the user in context we can assume we don't have it. Return 401.
		errorcode.NotAllowed.Render(w, r, http.StatusUnauthorized, "insufficient permissions to create a space.")
		return
	}

	client := g.GetGatewayClient()
	drive := libregraph.Drive{}
	if err := json.NewDecoder(r.Body).Decode(&drive); err != nil {
		logger.Debug().Err(err).Interface("body", r.Body).Msg("could not create drive: invalid body schema definition")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid body schema definition")
		return
	}
	spaceName := strings.TrimSpace(drive.Name)
	if err := validateSpaceName(spaceName); err != nil {
		logger.Debug().Str("name", spaceName).Err(err).Msg("could not create drive: name validation failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, fmt.Sprintf("invalid spacename: %s", err.Error()))
		return
	}

	var driveType string
	if drive.DriveType != nil {
		driveType = *drive.DriveType
	}
	switch driveType {
	case "", _spaceTypeProject:
		driveType = _spaceTypeProject
	default:
		logger.Debug().Str("type", driveType).Msg("could not create drive: drives of this type cannot be created via this api")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "drives of this type cannot be created via this api")
		return
	}

	csr := storageprovider.CreateStorageSpaceRequest{
		Type:  driveType,
		Name:  spaceName,
		Quota: getQuota(drive.Quota, g.config.Spaces.DefaultQuota),
	}

	if drive.Description != nil {
		csr.Opaque = utils.AppendPlainToOpaque(csr.Opaque, "description", *drive.Description)
	}

	if drive.DriveAlias != nil {
		csr.Opaque = utils.AppendPlainToOpaque(csr.Opaque, "spaceAlias", *drive.DriveAlias)
	}

	if driveType == _spaceTypePersonal {
		csr.Owner = us
	}

	resp, err := client.CreateStorageSpace(r.Context(), &csr)
	if err != nil {
		logger.Error().Err(err).Msg("could not create drive: transport error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	if resp.GetStatus().GetCode() != cs3rpc.Code_CODE_OK {
		if resp.GetStatus().GetCode() == cs3rpc.Code_CODE_PERMISSION_DENIED {
			logger.Debug().Str("grpcmessage", resp.GetStatus().GetMessage()).Msg("could not create drive: permission denied")
			errorcode.NotAllowed.Render(w, r, http.StatusForbidden, "permission denied")
			return
		}
		logger.Debug().Interface("grpcmessage", csr).Str("grpc", resp.GetStatus().GetMessage()).Msg("could not create drive: grpc error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, resp.GetStatus().GetMessage())
		return
	}

	webDavBaseURL, err := g.getWebDavBaseURL()
	if err != nil {
		logger.Error().Str("url", webDavBaseURL.String()).Err(err).Msg("could not create drive: error parsing webdav base url")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	newDrive, err := g.cs3StorageSpaceToDrive(r.Context(), webDavBaseURL, resp.StorageSpace)
	if err != nil {
		logger.Debug().Err(err).Msg("could not create drive: error parsing drive")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, newDrive)
}

// UpdateDrive updates the properties of a storage drive (space).
func (g Graph) UpdateDrive(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Info().Msg("calling update drive")
	driveID, err := url.PathUnescape(chi.URLParam(r, "driveID"))
	if err != nil {
		logger.Debug().Err(err).Str("id", chi.URLParam(r, "driveID")).Msg("could not update drive, unescaping drive id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping drive id failed")
		return
	}

	if driveID == "" {
		logger.Debug().Msg("Could not update drive, missing drive id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing drive id")
		return
	}

	drive := libregraph.Drive{}
	if err = json.NewDecoder(r.Body).Decode(&drive); err != nil {
		logger.Debug().Err(err).Interface("body", r.Body).Msg("could not update drive, invalid request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, fmt.Sprintf("invalid request body: error: %v", err.Error()))
		return
	}

	rid, err := storagespace.ParseID(driveID)
	if err != nil {
		logger.Debug().Err(err).Interface("id", rid).Msg("could not update drive, invalid resource id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid resource id")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	root := &rid

	client := g.GetGatewayClient()

	updateSpaceRequest := &storageprovider.UpdateStorageSpaceRequest{
		// Prepare the object to apply the diff from. The properties on StorageSpace will overwrite
		// the original storage space.
		StorageSpace: &storageprovider.StorageSpace{
			Id: &storageprovider.StorageSpaceId{
				OpaqueId: storagespace.FormatResourceID(rid),
			},
			Root: root,
		},
	}

	// Note: this is the Opaque prop of the request
	if restore, _ := strconv.ParseBool(r.Header.Get("restore")); restore {
		updateSpaceRequest.Opaque = &types.Opaque{
			Map: map[string]*types.OpaqueEntry{
				"restore": {
					Decoder: "plain",
					Value:   []byte("true"),
				},
			},
		}
	}

	if drive.Description != nil {
		updateSpaceRequest.StorageSpace.Opaque = utils.AppendPlainToOpaque(updateSpaceRequest.StorageSpace.Opaque, "description", *drive.Description)
	}

	if drive.DriveAlias != nil {
		updateSpaceRequest.StorageSpace.Opaque = utils.AppendPlainToOpaque(updateSpaceRequest.StorageSpace.Opaque, "spaceAlias", *drive.DriveAlias)
	}

	for _, special := range drive.Special {
		if special.Id != nil {
			updateSpaceRequest.StorageSpace.Opaque = utils.AppendPlainToOpaque(updateSpaceRequest.StorageSpace.Opaque, *special.SpecialFolder.Name, *special.Id)
		}
	}

	if drive.Name != "" {
		spacename := strings.TrimSpace(drive.Name)
		if err := validateSpaceName(spacename); err != nil {
			logger.Info().Err(err).Msg("could not update drive: spacename invalid")
			errorcode.GeneralException.Render(w, r, http.StatusBadRequest, err.Error())
			return
		}

		updateSpaceRequest.StorageSpace.Name = spacename
	}

	if drive.Quota.HasTotal() {
		user := revactx.ContextMustGetUser(r.Context())
		canSetSpaceQuota, err := g.canSetSpaceQuota(r.Context(), user)
		if err != nil {
			logger.Error().Err(err).Msg("could not update drive: failed to check if the user can set space quota")
			errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		if !canSetSpaceQuota {
			logger.Debug().
				Bool("cansetspacequota", canSetSpaceQuota).
				Msg("could not update drive: user is not allowed to set the space quota")
			errorcode.NotAllowed.Render(w, r, http.StatusUnauthorized, "user is not allowed to set the space quota")
			return
		}
		updateSpaceRequest.StorageSpace.Quota = &storageprovider.Quota{
			QuotaMaxBytes: uint64(*drive.Quota.Total),
		}
	}

	logger.Debug().Interface("payload", updateSpaceRequest).Msg("calling update space on backend")
	resp, err := client.UpdateStorageSpace(r.Context(), updateSpaceRequest)
	if err != nil {
		logger.Error().Err(err).Msg("could not update drive: transport error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "transport error")
		return
	}

	if resp.GetStatus().GetCode() != cs3rpc.Code_CODE_OK {
		switch resp.Status.GetCode() {
		case cs3rpc.Code_CODE_NOT_FOUND:
			logger.Debug().Interface("id", rid).Msg("could not update drive: drive not found")
			errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, resp.GetStatus().GetMessage())
			return
		case cs3rpc.Code_CODE_PERMISSION_DENIED:
			logger.Debug().Interface("id", rid).Msg("could not update drive, permission denied")
			errorcode.NotAllowed.Render(w, r, http.StatusForbidden, resp.GetStatus().GetMessage())
			return
		default:
			logger.Debug().Interface("id", rid).Str("grpc", resp.GetStatus().GetMessage()).Msg("could not update drive: grpc error")
			errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, resp.GetStatus().GetMessage())
			return
		}
	}

	webDavBaseURL, err := g.getWebDavBaseURL()
	if err != nil {
		logger.Error().Err(err).Interface("url", webDavBaseURL.String()).Msg("could not update drive: error parsing url")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	spaces, err := g.formatDrives(r.Context(), webDavBaseURL, []*storageprovider.StorageSpace{resp.StorageSpace})
	if err != nil {
		logger.Debug().Err(err).Msg("could not update drive: error parsing grpc response")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, spaces[0])
}

func (g Graph) formatDrives(ctx context.Context, baseURL *url.URL, storageSpaces []*storageprovider.StorageSpace) ([]*libregraph.Drive, error) {
	responses := make([]*libregraph.Drive, 0, len(storageSpaces))
	for _, storageSpace := range storageSpaces {
		res, err := g.cs3StorageSpaceToDrive(ctx, baseURL, storageSpace)
		if err != nil {
			return nil, err
		}

		// can't access disabled space
		if utils.ReadPlainFromOpaque(storageSpace.Opaque, "trashed") != "trashed" {
			res.Special = g.getExtendedSpaceProperties(ctx, baseURL, storageSpace)
		} else {
			res.Quota = nil
		}
		responses = append(responses, res)
	}

	return responses, nil
}

// ListStorageSpacesWithFilters List Storage Spaces using filters
func (g Graph) ListStorageSpacesWithFilters(ctx context.Context, filters []*storageprovider.ListStorageSpacesRequest_Filter, unrestricted bool) (*storageprovider.ListStorageSpacesResponse, error) {
	client := g.GetGatewayClient()

	permissions := make(map[string]struct{}, 1)
	s := settingssvc.NewPermissionService("com.owncloud.api.settings", grpc.DefaultClient())

	_, err := s.GetPermissionByID(ctx, &settingssvc.GetPermissionByIDRequest{
		PermissionId: settingsServiceExt.ListAllSpacesPermissionID,
	})

	// No error means the user has the permission
	if err == nil {
		permissions[settingsServiceExt.ListAllSpacesPermissionName] = struct{}{}
	}
	value, err := json.Marshal(permissions)
	if err != nil {
		return nil, err
	}
	lReq := &storageprovider.ListStorageSpacesRequest{
		Opaque: &types.Opaque{Map: map[string]*types.OpaqueEntry{
			"permissions": {
				Decoder: "json",
				Value:   value,
			},
			"unrestricted": {
				Decoder: "plain",
				Value:   []byte(strconv.FormatBool(unrestricted)),
			},
		}},
		Filters: filters,
	}
	res, err := client.ListStorageSpaces(ctx, lReq)
	return res, err
}

func (g Graph) cs3StorageSpaceToDrive(ctx context.Context, baseURL *url.URL, space *storageprovider.StorageSpace) (*libregraph.Drive, error) {
	logger := g.logger.SubloggerWithRequestID(ctx)
	if space.Root == nil {
		logger.Error().Msg("unable to parse space: space has no root")
		return nil, errors.New("space has no root")
	}
	spaceRid := *space.Root
	if space.Root.GetSpaceId() == space.Root.GetOpaqueId() {
		spaceRid.OpaqueId = ""
	}
	spaceID := storagespace.FormatResourceID(spaceRid)

	permissions := g.cs3PermissionsToLibreGraph(ctx, space)

	drive := &libregraph.Drive{
		Id:   libregraph.PtrString(spaceID),
		Name: space.Name,
		//"createdDateTime": "string (timestamp)", // TODO read from StorageSpace ... needs Opaque for now
		DriveType: &space.SpaceType,
		Root: &libregraph.DriveItem{
			Id:          libregraph.PtrString(storagespace.FormatResourceID(spaceRid)),
			Permissions: permissions,
		},
	}
	if space.SpaceType == "mountpoint" {
		var remoteItem *libregraph.RemoteItem
		grantID := storageprovider.ResourceId{
			StorageId: utils.ReadPlainFromOpaque(space.Opaque, "grantStorageID"),
			SpaceId:   utils.ReadPlainFromOpaque(space.Opaque, "grantSpaceID"),
			OpaqueId:  utils.ReadPlainFromOpaque(space.Opaque, "grantOpaqueID"),
		}
		if grantID.SpaceId != "" && grantID.OpaqueId != "" {
			var err error
			remoteItem, err = g.getRemoteItem(ctx, &grantID, baseURL)
			if err != nil {
				logger.Debug().Err(err).Interface("id", grantID).Msg("could not fetch remote item for space, continue")
			}
		}
		if remoteItem != nil {
			drive.Root.RemoteItem = remoteItem
		}
	}

	if space.Opaque != nil {
		if description, ok := space.Opaque.Map["description"]; ok {
			drive.Description = libregraph.PtrString(string(description.Value))
		}

		if alias, ok := space.Opaque.Map["spaceAlias"]; ok {
			drive.DriveAlias = libregraph.PtrString(string(alias.Value))
		}

		if v, ok := space.Opaque.Map["trashed"]; ok {
			deleted := &libregraph.Deleted{}
			deleted.SetState(string(v.Value))
			drive.Root.Deleted = deleted
		}

		if entry, ok := space.Opaque.Map["etag"]; ok {
			drive.Root.ETag = libregraph.PtrString(string(entry.Value))
		}
	}

	if baseURL != nil {
		webDavURL := *baseURL
		webDavURL.Path = path.Join(webDavURL.Path, spaceID)
		drive.Root.WebDavUrl = libregraph.PtrString(webDavURL.String())
	}

	webURL, err := url.Parse(g.config.Spaces.WebDavBase)
	if err != nil {
		logger.Error().
			Err(err).
			Str("url", g.config.Spaces.WebDavBase).
			Msg("failed to parse webURL base url")
		return nil, err
	}

	webURL.Path = path.Join(webURL.Path, "f", storagespace.FormatResourceID(spaceRid))
	drive.WebUrl = libregraph.PtrString(webURL.String())

	if space.Owner != nil && space.Owner.Id != nil {
		drive.Owner = &libregraph.IdentitySet{
			User: &libregraph.Identity{
				Id: &space.Owner.Id.OpaqueId,
				// DisplayName: , TODO read and cache from users provider
			},
		}
	}
	if space.Mtime != nil {
		lastModified := cs3TimestampToTime(space.Mtime)
		drive.LastModifiedDateTime = &lastModified
	}

	drive.Quota = &libregraph.Quota{}
	if space.Quota != nil {
		var t int64
		if space.Quota.QuotaMaxBytes > math.MaxInt64 {
			t = math.MaxInt64
		} else {
			t = int64(space.Quota.QuotaMaxBytes)
		}
		drive.Quota.Total = &t
	}

	return drive, nil
}

func (g Graph) cs3PermissionsToLibreGraph(ctx context.Context, space *storageprovider.StorageSpace) []libregraph.Permission {
	if space.Opaque == nil {
		return nil
	}
	logger := g.logger.SubloggerWithRequestID(ctx)

	var permissionsMap map[string]*storageprovider.ResourcePermissions
	opaqueGrants, ok := space.Opaque.Map["grants"]
	if ok {
		err := json.Unmarshal(opaqueGrants.Value, &permissionsMap)
		if err != nil {
			logger.Debug().
				Err(err).
				Interface("space", space.Root).
				Bytes("grants", opaqueGrants.Value).
				Msg("unable to parse space: failed to read spaces grants")
		}
	}
	if len(permissionsMap) == 0 {
		return nil
	}

	var permissionsExpirations map[string]*types.Timestamp
	opaqueGrantsExpirations, ok := space.Opaque.Map["grants_expirations"]
	if ok {
		err := json.Unmarshal(opaqueGrantsExpirations.Value, &permissionsExpirations)
		if err != nil {
			logger.Debug().
				Err(err).
				Interface("space", space.Root).
				Bytes("grants_expirations", opaqueGrantsExpirations.Value).
				Msg("unable to parse space: failed to read spaces grants expirations")
		}
	}

	var groupsMap map[string]struct{}
	opaqueGroups, ok := space.Opaque.Map["groups"]
	if ok {
		err := json.Unmarshal(opaqueGroups.Value, &groupsMap)
		if err != nil {
			logger.Debug().
				Err(err).
				Interface("space", space.Root).
				Bytes("groups", opaqueGroups.Value).
				Msg("unable to parse space: failed to read spaces groups")
		}
	}

	permissions := make([]libregraph.Permission, 0, len(permissionsMap))
	for id, perm := range permissionsMap {
		// This temporary variable is necessary since we need to pass a pointer to the
		// libregraph.Identity and if we pass the pointer from the loop every identity
		// will have the same id.
		tmp := id
		var identitySet libregraph.IdentitySet
		if _, ok := groupsMap[id]; ok {
			var group libregraph.Group
			if item := g.groupsCache.Get(id); item == nil {
				if requestedGroup, err := g.identityBackend.GetGroup(ctx, id, url.Values{}); err == nil {
					group = *requestedGroup
					g.groupsCache.Set(id, group, ttlcache.DefaultTTL)
				}
			} else {
				group = item.Value()
			}

			identitySet = libregraph.IdentitySet{Group: &libregraph.Identity{Id: &tmp, DisplayName: group.GetDisplayName()}}
		} else {
			var user libregraph.User
			if item := g.usersCache.Get(id); item == nil {
				if requestedUser, err := g.identityBackend.GetUser(ctx, id, &godata.GoDataRequest{}); err == nil {
					user = *requestedUser
					g.usersCache.Set(id, user, ttlcache.DefaultTTL)
				}
			} else {
				user = item.Value()
			}

			identitySet = libregraph.IdentitySet{User: &libregraph.Identity{Id: &tmp, DisplayName: user.GetDisplayName()}}
		}

		p := libregraph.Permission{
			GrantedToIdentities: []libregraph.IdentitySet{identitySet},
		}

		if exp := permissionsExpirations[id]; exp != nil {
			p.ExpirationDateTime = libregraph.PtrTime(time.Unix(int64(exp.GetSeconds()), int64(exp.GetNanos())))
		}

		// we need to map the permissions to the roles
		switch {
		// having RemoveGrant qualifies you as a manager
		case perm.RemoveGrant:
			p.SetRoles([]string{"manager"})
		// InitiateFileUpload means you are an editor
		case perm.InitiateFileUpload:
			p.SetRoles([]string{"editor"})
		// Stat permission at least makes you a viewer
		case perm.Stat:
			p.SetRoles([]string{"viewer"})
		}
		permissions = append(permissions, p)
	}
	return permissions
}

func (g Graph) getDriveQuota(ctx context.Context, space *storageprovider.StorageSpace) (libregraph.Quota, error) {
	logger := g.logger.SubloggerWithRequestID(ctx)
	client := g.GetGatewayClient()

	req := &gateway.GetQuotaRequest{
		Ref: &storageprovider.Reference{
			ResourceId: space.Root,
			Path:       ".",
		},
	}
	res, err := client.GetQuota(ctx, req)
	switch {
	case err != nil:
		logger.Error().Err(err).Interface("ref", req.Ref).Msg("could not call GetQuota: transport error")
		return libregraph.Quota{}, nil
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_UNIMPLEMENTED:
		logger.Debug().Msg("get quota is not implemented on the storage driver")
		return libregraph.Quota{}, nil
	case res.GetStatus().GetCode() != cs3rpc.Code_CODE_OK:
		logger.Debug().Str("grpc", res.GetStatus().GetMessage()).Msg("error sending get quota grpc request")
		return libregraph.Quota{}, errors.New(res.GetStatus().GetMessage())
	}

	var remaining int64
	if res.Opaque != nil {
		m := res.Opaque.Map
		if e, ok := m["remaining"]; ok {
			remaining, _ = strconv.ParseInt(string(e.Value), 10, 64)
		}
	}

	used := int64(res.UsedBytes)
	total := int64(res.TotalBytes)
	qta := libregraph.Quota{
		Remaining: &remaining,
		Used:      &used,
		Total:     &total,
	}

	var t int64
	if total != 0 {
		t = total
	} else {
		// Quota was not set
		// Use remaining bytes to calculate state
		t = remaining
	}
	state := calculateQuotaState(t, used)
	qta.State = &state

	return qta, nil
}

func calculateQuotaState(total int64, used int64) (state string) {
	percent := (float64(used) / float64(total)) * 100

	switch {
	case percent <= float64(75):
		return "normal"
	case percent <= float64(90):
		return "nearing"
	case percent <= float64(99):
		return "critical"
	default:
		return "exceeded"
	}
}

func getQuota(quota *libregraph.Quota, defaultQuota string) *storageprovider.Quota {
	switch {
	case quota != nil && quota.Total != nil:
		if q := *quota.Total; q >= 0 {
			return &storageprovider.Quota{QuotaMaxBytes: uint64(q)}
		}
		fallthrough
	case defaultQuota != "":
		if q, err := strconv.ParseInt(defaultQuota, 10, 64); err == nil && q >= 0 {
			return &storageprovider.Quota{QuotaMaxBytes: uint64(q)}
		}
		fallthrough
	default:
		return nil
	}
}

func (g Graph) canSetSpaceQuota(ctx context.Context, user *userv1beta1.User) (bool, error) {
	_, err := g.permissionsService.GetPermissionByID(ctx, &settingssvc.GetPermissionByIDRequest{PermissionId: settingsServiceExt.SetSpaceQuotaPermissionID})
	if err != nil {
		merror := merrors.FromError(err)
		if merror.Status == http.StatusText(http.StatusNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func generateCs3Filters(request *godata.GoDataRequest) ([]*storageprovider.ListStorageSpacesRequest_Filter, error) {
	var filters []*storageprovider.ListStorageSpacesRequest_Filter
	if request.Query.Filter != nil {
		if request.Query.Filter.Tree.Token.Value == "eq" {
			switch request.Query.Filter.Tree.Children[0].Token.Value {
			case "driveType":
				filters = append(filters, listStorageSpacesTypeFilter(strings.Trim(request.Query.Filter.Tree.Children[1].Token.Value, "'")))
			case "id":
				filters = append(filters, listStorageSpacesIDFilter(strings.Trim(request.Query.Filter.Tree.Children[1].Token.Value, "'")))
			}
		} else {
			err := errors.Errorf("unsupported filter operand: %s", request.Query.Filter.Tree.Token.Value)
			return nil, err
		}
	}
	return filters, nil
}

func listStorageSpacesIDFilter(id string) *storageprovider.ListStorageSpacesRequest_Filter {
	return &storageprovider.ListStorageSpacesRequest_Filter{
		Type: storageprovider.ListStorageSpacesRequest_Filter_TYPE_ID,
		Term: &storageprovider.ListStorageSpacesRequest_Filter_Id{
			Id: &storageprovider.StorageSpaceId{
				OpaqueId: id,
			},
		},
	}
}

func listStorageSpacesUserFilter(id string) *storageprovider.ListStorageSpacesRequest_Filter {
	return &storageprovider.ListStorageSpacesRequest_Filter{
		Type: storageprovider.ListStorageSpacesRequest_Filter_TYPE_USER,
		Term: &storageprovider.ListStorageSpacesRequest_Filter_User{
			User: &userv1beta1.UserId{
				OpaqueId: id,
			},
		},
	}
}

func listStorageSpacesTypeFilter(spaceType string) *storageprovider.ListStorageSpacesRequest_Filter {
	return &storageprovider.ListStorageSpacesRequest_Filter{
		Type: storageprovider.ListStorageSpacesRequest_Filter_TYPE_SPACE_TYPE,
		Term: &storageprovider.ListStorageSpacesRequest_Filter_SpaceType{
			SpaceType: spaceType,
		},
	}
}

// DeleteDrive deletes a storage drive (space).
func (g Graph) DeleteDrive(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Info().Msg("calling delete drive")
	driveID, err := url.PathUnescape(chi.URLParam(r, "driveID"))
	if err != nil {
		logger.Debug().Err(err).Str("id", chi.URLParam(r, "driveID")).Msg("could not delete drive: unescaping drive id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping drive id failed")
		return
	}

	if driveID == "" {
		logger.Debug().Msg("could not delete drive: missing drive id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing drive id")
		return
	}

	rid, err := storagespace.ParseID(driveID)
	if err != nil {
		logger.Debug().Interface("id", rid).Err(err).Msg("could not delete drive: invalid resource id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid resource id")
		return
	}

	purge := parsePurgeHeader(r.Header)

	var opaque *types.Opaque
	if purge {
		opaque = &types.Opaque{
			Map: map[string]*types.OpaqueEntry{
				"purge": {},
			},
		}
	}

	dRes, err := g.gatewayClient.DeleteStorageSpace(r.Context(), &storageprovider.DeleteStorageSpaceRequest{
		Opaque: opaque,
		Id: &storageprovider.StorageSpaceId{
			OpaqueId: storagespace.FormatResourceID(rid),
		},
	})
	if err != nil {
		logger.Error().Err(err).Interface("id", rid).Msg("could not delete drive: transport error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "transport error")
		return
	}

	switch dRes.GetStatus().GetCode() {
	case cs3rpc.Code_CODE_OK:
		w.WriteHeader(http.StatusNoContent)
		return
	case cs3rpc.Code_CODE_INVALID_ARGUMENT:
		logger.Debug().Interface("id", rid).Str("grpc", dRes.GetStatus().GetMessage()).Msg("could not delete drive: invalid argument")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, dRes.Status.Message)
		return
	case cs3rpc.Code_CODE_PERMISSION_DENIED:
		logger.Debug().Interface("id", rid).Msg("could not delete drive: permission denied")
		errorcode.NotAllowed.Render(w, r, http.StatusForbidden, "permission denied to delete drive")
		return
	// don't expose internal error codes to the outside world
	default:
		logger.Debug().Str("grpc", dRes.GetStatus().GetMessage()).Interface("id", rid).Msg("could not delete drive: grpc error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "grpc error")
		return
	}

}

func sortSpaces(req *godata.GoDataRequest, spaces []*libregraph.Drive) ([]*libregraph.Drive, error) {
	if req.Query.OrderBy == nil || len(req.Query.OrderBy.OrderByItems) != 1 {
		return spaces, nil
	}
	var less func(i, j int) bool

	switch req.Query.OrderBy.OrderByItems[0].Field.Value {
	case "name":
		less = func(i, j int) bool {
			return strings.ToLower(spaces[i].GetName()) < strings.ToLower(spaces[j].GetName())
		}
	case "lastModifiedDateTime":
		less = func(i, j int) bool {
			return lessSpacesByLastModifiedDateTime(spaces[i], spaces[j])
		}
	default:
		return nil, errors.Errorf("we do not support <%s> as a order parameter", req.Query.OrderBy.OrderByItems[0].Field.Value)
	}

	if req.Query.OrderBy.OrderByItems[0].Order == _sortDescending {
		sort.Slice(spaces, reverse(less))
	} else {
		sort.Slice(spaces, less)
	}
	return spaces, nil
}

func validateSpaceName(name string) error {
	if name == "" {
		return ErrNameEmpty
	}

	if len(name) > _maxSpaceNameLength {
		return ErrNameTooLong
	}

	for _, c := range _invalidSpaceNameCharacters {
		if strings.Contains(name, c) {
			return ErrForbiddenCharacter
		}
	}

	return nil
}
