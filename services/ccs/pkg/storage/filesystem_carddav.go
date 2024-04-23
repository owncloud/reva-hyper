package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	providerv1beta1 "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	revaContext "github.com/cs3org/reva/v2/pkg/ctx"
	"github.com/cs3org/reva/v2/pkg/storage/utils/metadata"
	"github.com/cs3org/reva/v2/pkg/utils"
	"net/http"
	"path"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/DeepDiver1975/go-webdav"
	"github.com/DeepDiver1975/go-webdav/carddav"
	"github.com/emersion/go-vcard"
)

const addressBookFileName = "addressbook.json"

func (b *filesystemBackend) AddressBookHomeSetPath(ctx context.Context) (string, error) {
	user, ok := revaContext.ContextGetUser(ctx)
	if !ok {
		return "", errors.New("no user in context")
	}
	return fmt.Sprintf("/ccs/addressbooks/%s/", user.Username), nil
}

func (b *filesystemBackend) CreateAddressBook(ctx context.Context, addressBook *carddav.AddressBook) error {
	resourceName := path.Base(addressBook.Path)
	localPath, err := b.localCardDAVDir(ctx, resourceName)
	if err != nil {
		return fmt.Errorf("error creating addressbook calendar: %s", err.Error())
	}

	log.Debug().Str("local", localPath).Str("url", addressBook.Path).Msg("filesystem.CreateAddressBook()")

	blob, err := json.MarshalIndent(addressBook, "", "  ")
	if err != nil {
		return fmt.Errorf("error creating addressbook: %s", err.Error())
	}
	req := metadata.UploadRequest{
		Path:        path.Join(localPath, addressBookFileName),
		Content:     blob,
		IfNoneMatch: []string{"*"},
	}
	_, err = b.storage.Upload(ctx, req)

	if err != nil {
		if isAlreadyExists(err) {
			return webdav.NewHTTPError(405, errors.New("the resource you tried to create already exists"))
		}

		return fmt.Errorf("error writing addressbook: %s", err.Error())
	}
	return nil
}

func (b *filesystemBackend) DeleteAddressBook(ctx context.Context, path string) error {
	log.Debug().Str("path", path).Msg("filesystem.DeleteAddressBook()")

	localPath, err := b.safeLocalCardDAVPath(ctx, path)
	if err != nil {
		return err
	}
	return b.storage.Delete(ctx, localPath)
}

func (b *filesystemBackend) localCardDAVDir(ctx context.Context, components ...string) (string, error) {
	homeSetPath, err := b.AddressBookHomeSetPath(ctx)
	if err != nil {
		return "", err
	}

	return b.localDir(ctx, homeSetPath, components...)
}

func (b *filesystemBackend) safeLocalCardDAVPath(ctx context.Context, urlPath string) (string, error) {
	homeSetPath, err := b.AddressBookHomeSetPath(ctx)
	if err != nil {
		return "", err
	}

	return b.safeLocalPath(ctx, homeSetPath, urlPath)
}

func vcardPropFilter(card vcard.Card, props []string) vcard.Card {
	if card == nil {
		return nil
	}

	if len(props) == 0 {
		return card
	}

	result := make(vcard.Card)
	result["VERSION"] = card["VERSION"]
	for _, prop := range props {
		value, ok := card[prop]
		if ok {
			result[prop] = value
		}
	}

	return result
}

func (b *filesystemBackend) vcardFromFile(ctx context.Context, path string, propFilter []string) (vcard.Card, string, error) {
	req := metadata.DownloadRequest{
		Path: path,
	}
	response, err := b.storage.Download(ctx, req)
	if err != nil {
		return nil, "", err
	}

	r := bytes.NewReader(response.Content)
	dec := vcard.NewDecoder(r)
	card, err := dec.Decode()
	if err != nil {
		return nil, "", err
	}

	return vcardPropFilter(card, propFilter), response.Etag, nil
}

func (b *filesystemBackend) loadAllAddressObjects(ctx context.Context, urlPath string, propFilter []string) ([]carddav.AddressObject, error) {
	var result []carddav.AddressObject

	localPath, err := b.safeLocalCardDAVPath(ctx, urlPath)
	if err != nil {
		return result, err
	}

	log.Debug().Str("path", localPath).Msg("loading address objects")

	dir, err := b.storage.ListDir(ctx, localPath)
	if err != nil {
		return nil, err
	}
	for _, f := range dir {
		// Skip address book meta data files
		if f.Type != providerv1beta1.ResourceType_RESOURCE_TYPE_FILE || filepath.Ext(f.Name) != ".vcf" {
			continue
		}

		calPath := filepath.Join(localPath, f.Path)
		card, _, err := b.vcardFromFile(ctx, calPath, propFilter)
		if err != nil {
			fmt.Printf("load event error for %s: %v\n", f.Path, err)
			// TODO: return err ???
			continue
		}

		obj := carddav.AddressObject{
			Path:          path.Join(urlPath, f.Name),
			ModTime:       utils.TSToTime(f.Mtime),
			ContentLength: int64(f.Size),
			ETag:          f.Etag,
			Card:          card,
		}
		result = append(result, obj)
	}

	return result, err
}

func (b *filesystemBackend) ListAddressBooks(ctx context.Context) ([]carddav.AddressBook, error) {
	log.Debug().Msg("filesystem.ListAddressBooks()")

	localPath, err := b.localCardDAVDir(ctx)
	if err != nil {
		return nil, err
	}

	log.Debug().Str("path", localPath).Msg("looking for address books")

	var result []carddav.AddressBook

	dir, err := b.storage.ListDir(ctx, localPath)
	if err != nil {
		return nil, err
	}
	for _, f := range dir {
		if f.Type != providerv1beta1.ResourceType_RESOURCE_TYPE_CONTAINER || f.Path == localPath {
			continue
		}
		calPath := path.Join(f.Path, addressBookFileName)
		addressbook, err := b.readAddressbook(ctx, calPath)
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

		result = append(result, *addressbook)
	}

	log.Debug().Int("results", len(result)).Err(err).Msg("filesystem.ListAddressBooks() done")
	return result, err
}

func (b *filesystemBackend) GetAddressBook(ctx context.Context, urlPath string) (*carddav.AddressBook, error) {
	log.Debug().Str("path", urlPath).Msg("filesystem.AddressBook()")

	localPath, err := b.safeLocalCardDAVPath(ctx, urlPath)
	if err != nil {
		return nil, err
	}
	localPath = filepath.Join(localPath, addressBookFileName)

	log.Debug().Str("path", localPath).Msg("loading addressbook")

	return b.readAddressbook(ctx, localPath)
}

func (b *filesystemBackend) readAddressbook(ctx context.Context, localPath string) (*carddav.AddressBook, error) {
	data, err := b.storage.SimpleDownload(ctx, localPath)
	if err != nil {
		if isNotFound(err) {
			return nil, webdav.NewHTTPError(404, errors.New("not found"))
		}
		return nil, fmt.Errorf("error opening address book: %s", err.Error())
	}
	var addressBook carddav.AddressBook
	err = json.Unmarshal(data, &addressBook)
	if err != nil {
		return nil, fmt.Errorf("error reading address book: %s", err.Error())
	}

	return &addressBook, nil
}

func (b *filesystemBackend) GetAddressObject(ctx context.Context, objPath string, req *carddav.AddressDataRequest) (*carddav.AddressObject, error) {
	log.Debug().Str("path", objPath).Msg("filesystem.GetAddressObject()")

	localPath, err := b.safeLocalCardDAVPath(ctx, objPath)
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
	if req != nil && !req.AllProp {
		propFilter = req.Props
	}

	card, etag, err := b.vcardFromFile(ctx, localPath, propFilter)
	if err != nil {
		log.Debug().Str("path", localPath).Err(err).Msg("error reading addressbook")
		return nil, err
	}

	obj := carddav.AddressObject{
		Path:          objPath,
		ModTime:       utils.TSToTime(info.Mtime),
		ContentLength: int64(info.Size),
		ETag:          etag,
		Card:          card,
	}
	return &obj, nil
}

func (b *filesystemBackend) ListAddressObjects(ctx context.Context, urlPath string, req *carddav.AddressDataRequest) ([]carddav.AddressObject, error) {
	log.Debug().Str("path", urlPath).Msg("filesystem.ListAddressObjects()")

	var propFilter []string
	if req != nil && !req.AllProp {
		propFilter = req.Props
	}

	result, err := b.loadAllAddressObjects(ctx, urlPath, propFilter)
	log.Debug().Int("results", len(result)).Err(err).Msg("filesystem.ListAddressObjects() done")
	return result, err
}

func (b *filesystemBackend) QueryAddressObjects(ctx context.Context, urlPath string, query *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
	log.Debug().Str("path", urlPath).Msg("filesystem.QueryAddressObjects()")

	var propFilter []string
	if query != nil && !query.DataRequest.AllProp {
		propFilter = query.DataRequest.Props
	}

	result, err := b.loadAllAddressObjects(ctx, urlPath, propFilter)
	log.Debug().Int("results", len(result)).Err(err).Msg("filesystem.QueryAddressObjects() load done")
	if err != nil {
		return result, err
	}

	filtered, err := carddav.Filter(query, result)
	log.Debug().Int("results", len(filtered)).Err(err).Msg("filesystem.QueryAddressObjects() filter done")
	return filtered, err
}

func (b *filesystemBackend) PutAddressObject(ctx context.Context, objPath string, card vcard.Card, opts *carddav.PutAddressObjectOptions) (*carddav.AddressObject, error) {
	log.Debug().Str("path", objPath).Msg("filesystem.PutAddressObject()")

	// TODO: validate carddav ???
	localPath, err := b.safeLocalCardDAVPath(ctx, objPath)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	enc := vcard.NewEncoder(&buf)
	err = enc.Encode(card)
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

	return b.GetAddressObject(ctx, objPath, nil)
}

func (b *filesystemBackend) DeleteAddressObject(ctx context.Context, path string) error {
	log.Debug().Str("path", path).Msg("filesystem.DeleteAddressObject()")

	localPath, err := b.safeLocalCardDAVPath(ctx, path)
	if err != nil {
		return err
	}
	return b.storage.Delete(ctx, localPath)
}
