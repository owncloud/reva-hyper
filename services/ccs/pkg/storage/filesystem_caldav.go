package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DeepDiver1975/go-webdav"
	"github.com/DeepDiver1975/go-webdav/caldav"
	providerv1beta1 "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	revaContext "github.com/cs3org/reva/v2/pkg/ctx"
	"github.com/cs3org/reva/v2/pkg/storage/utils/metadata"
	"github.com/cs3org/reva/v2/pkg/utils"
	"github.com/emersion/go-ical"
	"github.com/rs/zerolog/log"
	"net/http"
	"path"
	"path/filepath"
)

const calendarFileName = "calendar.json"

func (b *filesystemBackend) CalendarHomeSetPath(ctx context.Context) (string, error) {
	user, ok := revaContext.ContextGetUser(ctx)
	if !ok {
		return "", errors.New("no user in context")
	}
	return fmt.Sprintf("/ccs/calendars/%s/", user.Username), nil
}

func (b *filesystemBackend) CreateCalendar(ctx context.Context, calendar *caldav.Calendar) error {
	resourceName := path.Base(calendar.Path)
	localPath, err := b.localCalDAVDir(ctx, resourceName)
	if err != nil {
		return fmt.Errorf("error creating calendar: %s", err.Error())
	}

	log.Debug().Str("local", localPath).Str("url", calendar.Path).Msg("filesystem.CreateCalendar()")

	blob, err := json.MarshalIndent(calendar, "", "  ")
	if err != nil {
		return fmt.Errorf("error creating calendar: %s", err.Error())
	}
	req := metadata.UploadRequest{
		Path:        path.Join(localPath, calendarFileName),
		Content:     blob,
		IfNoneMatch: []string{"*"},
	}
	_, err = b.storage.Upload(ctx, req)
	if err != nil {
		if isAlreadyExists(err) {
			return webdav.NewHTTPError(405, errors.New("the resource you tried to create already exists"))
		}

		return fmt.Errorf("error writing calendar: %s", err.Error())
	}
	return nil
}

func (b *filesystemBackend) localCalDAVDir(ctx context.Context, components ...string) (string, error) {
	homeSetPath, err := b.CalendarHomeSetPath(ctx)
	if err != nil {
		return "", err
	}

	return b.localDir(ctx, homeSetPath, components...)
}

func (b *filesystemBackend) safeLocalCalDAVPath(ctx context.Context, urlPath string) (string, error) {
	homeSetPath, err := b.CalendarHomeSetPath(ctx)
	if err != nil {
		return "", err
	}

	return b.safeLocalPath(ctx, homeSetPath, urlPath)
}

func (b *filesystemBackend) calendarFromFile(ctx context.Context, path string, propFilter []string) (*ical.Calendar, string, error) {
	req := metadata.DownloadRequest{
		Path: path,
	}
	response, err := b.storage.Download(ctx, req)
	if err != nil {
		return nil, "", err
	}

	r := bytes.NewReader(response.Content)
	dec := ical.NewDecoder(r)
	cal, err := dec.Decode()
	if err != nil {
		return nil, "", err
	}

	return cal, response.Etag, nil
	// TODO implement
	//return icalPropFilter(cal, propFilter), nil
}

func (b *filesystemBackend) loadAllCalendarObjects(ctx context.Context, urlPath string, propFilter []string) ([]caldav.CalendarObject, error) {
	var result []caldav.CalendarObject

	localPath, err := b.safeLocalCalDAVPath(ctx, urlPath)
	if err != nil {
		return result, err
	}

	log.Debug().Str("path", localPath).Msg("loading calendar objects")

	dir, err := b.storage.ListDir(ctx, localPath)
	if err != nil {
		return nil, err
	}
	for _, f := range dir {
		// Skip address book meta data files
		if f.Type != providerv1beta1.ResourceType_RESOURCE_TYPE_FILE || filepath.Ext(f.Name) != ".ics" {
			continue
		}

		calPath := filepath.Join(localPath, f.Path)
		cal, _, err := b.calendarFromFile(ctx, calPath, propFilter)
		if err != nil {
			fmt.Printf("load calendar error for %s: %v\n", f.Path, err)
			// TODO: return err ???
			continue
		}

		obj := caldav.CalendarObject{
			Path:          path.Join(urlPath, f.Name),
			ModTime:       utils.TSToTime(f.Mtime),
			ContentLength: int64(f.Size),
			ETag:          f.Etag,
			Data:          cal,
		}
		result = append(result, obj)
	}

	return result, err
}

func (b *filesystemBackend) ListCalendars(ctx context.Context) ([]caldav.Calendar, error) {
	log.Debug().Msg("filesystem.ListCalendars()")

	localPath, err := b.localCalDAVDir(ctx)
	if err != nil {
		return nil, err
	}

	log.Debug().Str("path", localPath).Msg("looking for calendars")

	var result []caldav.Calendar

	dir, err := b.storage.ListDir(ctx, localPath)
	if err != nil {
		return nil, err
	}
	for _, f := range dir {
		if f.Type != providerv1beta1.ResourceType_RESOURCE_TYPE_CONTAINER || f.Path == localPath {
			continue
		}
		calPath := path.Join(f.Path, calendarFileName)
		calendar, err := b.readCalendar(ctx, calPath)
		if err != nil {
			// TODO: how to handle
			/*
				if os.IsNotExist(err) {
					return nil // not a calendar dir
				} else {
					return fmt.Errorf("error accessing %s: %s", calPath, err.Error())
				}
			*/
			continue
		}

		result = append(result, *calendar)
	}

	log.Debug().Int("results", len(result)).Err(err).Msg("filesystem.ListCalendars() done")
	return result, err
}

func (b *filesystemBackend) GetCalendar(ctx context.Context, urlPath string) (*caldav.Calendar, error) {
	log.Debug().Str("path", urlPath).Msg("filesystem.GetCalendar()")

	localPath, err := b.safeLocalCalDAVPath(ctx, urlPath)
	if err != nil {
		return nil, err
	}
	localPath = filepath.Join(localPath, calendarFileName)

	log.Debug().Str("path", localPath).Msg("loading calendar")

	return b.readCalendar(ctx, localPath)
}

func (b *filesystemBackend) readCalendar(ctx context.Context, localPath string) (*caldav.Calendar, error) {
	data, err := b.storage.SimpleDownload(ctx, localPath)
	if err != nil {
		if isNotFound(err) {
			return nil, webdav.NewHTTPError(404, errors.New("not found"))
		}
		return nil, fmt.Errorf("error opening calendar: %s", err.Error())
	}
	var calendar caldav.Calendar
	err = json.Unmarshal(data, &calendar)
	if err != nil {
		return nil, fmt.Errorf("error reading calendar: %s", err.Error())
	}

	return &calendar, nil
}

func (b *filesystemBackend) GetCalendarObject(ctx context.Context, objPath string, req *caldav.CalendarCompRequest) (*caldav.CalendarObject, error) {
	log.Debug().Str("path", objPath).Msg("filesystem.GetCalendarObject()")

	localPath, err := b.safeLocalCalDAVPath(ctx, objPath)
	if err != nil {
		return nil, err
	}

	info, err := b.storage.Stat(ctx, localPath)
	if err != nil {
		if isNotFound(err) {
			return nil, webdav.NewHTTPError(404, errors.New("not found"))
		}
		return nil, err
	}

	var propFilter []string
	if req != nil && !req.AllProps {
		propFilter = req.Props
	}

	calendar, etag, err := b.calendarFromFile(ctx, localPath, propFilter)
	if err != nil {
		log.Debug().Str("path", localPath).Err(err).Msg("error reading calendar")
		return nil, err
	}

	obj := caldav.CalendarObject{
		Path:          objPath,
		ModTime:       utils.TSToTime(info.Mtime),
		ContentLength: int64(info.Size),
		ETag:          etag,
		Data:          calendar,
	}
	return &obj, nil
}

func (b *filesystemBackend) ListCalendarObjects(ctx context.Context, urlPath string, req *caldav.CalendarCompRequest) ([]caldav.CalendarObject, error) {
	log.Debug().Str("path", urlPath).Msg("filesystem.ListCalendarObjects()")

	var propFilter []string
	if req != nil && !req.AllProps {
		propFilter = req.Props
	}

	result, err := b.loadAllCalendarObjects(ctx, urlPath, propFilter)
	log.Debug().Int("results", len(result)).Err(err).Msg("filesystem.ListCalendarObjects() done")
	return result, err
}

func (b *filesystemBackend) QueryCalendarObjects(ctx context.Context, urlPath string, query *caldav.CalendarQuery) ([]caldav.CalendarObject, error) {
	log.Debug().Str("path", urlPath).Msg("filesystem.QueryCalendarObjects()")

	var propFilter []string
	if query != nil && !query.CompRequest.AllProps {
		propFilter = query.CompRequest.Props
	}

	result, err := b.loadAllCalendarObjects(ctx, urlPath, propFilter)
	log.Debug().Int("results", len(result)).Err(err).Msg("filesystem.QueryCalendarObjects() load done")
	if err != nil {
		return result, err
	}

	filtered, err := caldav.Filter(query, result)
	log.Debug().Int("results", len(filtered)).Err(err).Msg("filesystem.QueryCalendarObjects() filter done")
	return filtered, err
}

func (b *filesystemBackend) PutCalendarObject(ctx context.Context, objPath string, calendar *ical.Calendar, opts *caldav.PutCalendarObjectOptions) (*caldav.CalendarObject, error) {
	log.Debug().Str("path", objPath).Msg("filesystem.PutCalendarObject()")

	_, _, err := caldav.ValidateCalendarObject(calendar)
	if err != nil {
		return nil, caldav.NewPreconditionError(caldav.PreconditionValidCalendarObjectResource)
	}

	localPath, err := b.safeLocalCalDAVPath(ctx, objPath)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	enc := ical.NewEncoder(&buf)
	err = enc.Encode(calendar)
	if err != nil {
		return nil, err
	}

	req := metadata.UploadRequest{
		Path:    localPath,
		Content: buf.Bytes(),
	}

	// TODO handle IfNoneMatch == ETag
	if opts.IfNoneMatch.IsWildcard() {
		// Make sure we're not overwriting an existing file
		req.IfNoneMatch = []string{"*"}
	} else if opts.IfMatch.IsWildcard() {
		// Make sure we _are_ overwriting an existing file
		// TODO: not existing in UploadRequest
		// req.IfMatch = []string{"*"}
	} else if opts.IfMatch.IsSet() {
		want, err := opts.IfMatch.ETag()
		if err != nil {
			return nil, webdav.NewHTTPError(http.StatusBadRequest, err)
		}
		req.IfMatchEtag = want
	}

	_, err = b.storage.Upload(ctx, req)
	if err != nil {
		return nil, err
	}
	calendarObject, err := b.GetCalendarObject(ctx, objPath, nil)
	if err != nil {
		return nil, err
	}

	return calendarObject, nil
}

func (b *filesystemBackend) DeleteCalendarObject(ctx context.Context, path string) error {
	log.Debug().Str("path", path).Msg("filesystem.DeleteCalendarObject()")

	localPath, err := b.safeLocalCalDAVPath(ctx, path)
	if err != nil {
		return err
	}
	return b.storage.Delete(ctx, localPath)
}
