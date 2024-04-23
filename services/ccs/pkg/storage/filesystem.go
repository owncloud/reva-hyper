package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/cs3org/reva/v2/pkg/errtypes"
	"github.com/cs3org/reva/v2/pkg/storage/utils/metadata"
	"path"
	"path/filepath"
	"strings"

	"github.com/DeepDiver1975/go-webdav"
	"github.com/DeepDiver1975/go-webdav/caldav"
	"github.com/DeepDiver1975/go-webdav/carddav"
)

type filesystemBackend struct {
	webdav.UserPrincipalBackend
	caldavPrefix  string
	carddavPrefix string
	storage       metadata.Storage
}

func isNotFound(err error) bool {
	var notFound errtypes.NotFound
	return errors.As(err, &notFound)
}
func isAlreadyExists(err error) bool {
	var notFound errtypes.AlreadyExists
	return errors.As(err, &notFound)
}

func NewFilesystem(storage metadata.Storage, caldavPrefix, carddavPrefix string, userPrincipalBackend webdav.UserPrincipalBackend) (caldav.Backend, carddav.Backend, error) {
	backend := &filesystemBackend{
		UserPrincipalBackend: userPrincipalBackend,
		caldavPrefix:         caldavPrefix,
		carddavPrefix:        carddavPrefix,
		storage:              storage,
	}
	return backend, backend, nil
}

func (b *filesystemBackend) ensureLocalDir(ctx context.Context, path string) error {
	segments := strings.Split(path, "/")
	cwd := "/"
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		cwd = filepath.Join(cwd, segment)
		err := b.storage.MakeDirIfNotExist(ctx, cwd)
		if err != nil {
			return fmt.Errorf("error creating '%s': %s", cwd, err.Error())
		}
	}
	return nil
}

func (b *filesystemBackend) localDir(ctx context.Context, homeSetPath string, components ...string) (string, error) {
	c := append([]string{homeSetPath}, components...)
	localPath := filepath.Join(c...)
	if err := b.ensureLocalDir(ctx, localPath); err != nil {
		return "", err
	}
	return localPath, nil
}

// don't use this directly, use localCalDAVPath or localCardDAVPath instead.
// note that homesetpath is expected to end in /
func (b *filesystemBackend) safeLocalPath(ctx context.Context, homeSetPath string, urlPath string) (string, error) {
	if err := b.ensureLocalDir(ctx, homeSetPath); err != nil {
		return "", err
	}

	if urlPath == "" {
		return homeSetPath, nil
	}

	// We are mapping to local filesystem path, so be conservative about what to accept
	if strings.HasSuffix(urlPath, "/") {
		urlPath = path.Clean(urlPath) + "/"
	} else {
		urlPath = path.Clean(urlPath)
	}
	if !strings.HasPrefix(urlPath, homeSetPath) {
		err := fmt.Errorf("access to resource outside of home set: %s", urlPath)
		return "", webdav.NewHTTPError(403, err)
	}
	urlPath = strings.TrimPrefix(urlPath, homeSetPath)

	dir, file := path.Split(urlPath)
	return filepath.Join(homeSetPath, dir, file), nil
}
