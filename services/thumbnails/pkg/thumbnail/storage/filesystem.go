package storage

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/owncloud/ocis/v2/ocis-pkg/log"
	"github.com/owncloud/ocis/v2/services/thumbnails/pkg/config"
)

const (
	filesDir = "files"
)

// NewFileSystemStorage creates a new instance of FileSystem
func NewFileSystemStorage(cfg config.FileSystemStorage, logger log.Logger) FileSystem {
	return FileSystem{
		root:   cfg.RootDirectory,
		logger: logger,
	}
}

// FileSystem represents a storage for the thumbnails using the local file system.
type FileSystem struct {
	root   string
	logger log.Logger
}

func (s FileSystem) Stat(key string) bool {
	img := filepath.Join(s.root, filesDir, key)
	if _, err := os.Stat(img); err != nil {
		return false
	}
	return true
}

func (s FileSystem) Get(key string) ([]byte, error) {
	img := filepath.Join(s.root, filesDir, key)
	content, err := os.ReadFile(img)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			s.logger.Debug().Str("err", err.Error()).Str("key", key).Msg("could not load thumbnail from store")
		}
		return nil, err
	}
	return content, nil
}

func (s FileSystem) Put(key string, img []byte) error {
	imgPath := filepath.Join(s.root, filesDir, key)
	dir := filepath.Dir(imgPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return errors.Wrapf(err, "error while creating directory %s", dir)
	}

	if _, err := os.Stat(imgPath); os.IsNotExist(err) {
		f, err := os.Create(imgPath)
		if err != nil {
			return errors.Wrapf(err, "could not create file \"%s\"", key)
		}
		defer f.Close()

		if _, err = f.Write(img); err != nil {
			return errors.Wrapf(err, "could not write to file \"%s\"", key)
		}
	}

	return nil
}

// BuildKey generate the unique key for a thumbnail.
// The key is structure as follows:
//
// <first two letters of checksum>/<next two letters of checksum>/<rest of checksum>/<width>x<height>.<filetype>
//
// e.g. 97/9f/4c8db98f7b82e768ef478d3c8612/500x300.png
//
// The key also represents the path to the thumbnail in the filesystem under the configured root directory.
func (s FileSystem) BuildKey(r Request) string {
	checksum := r.Checksum
	filetype := r.Types[0]

	parts := []string{strconv.Itoa(r.Resolution.Dx()), "x", strconv.Itoa(r.Resolution.Dy())}

	if r.Characteristic != "" {
		parts = append(parts, "-", r.Characteristic)
	}

	parts = append(parts, ".", filetype)

	return filepath.Join(checksum[:2], checksum[2:4], checksum[4:], strings.Join(parts, ""))
}
