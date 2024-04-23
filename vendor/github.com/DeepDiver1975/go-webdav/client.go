package webdav

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/DeepDiver1975/go-webdav/internal"
)

// HTTPClient performs HTTP requests. It's implemented by *http.Client.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type basicAuthHTTPClient struct {
	c                  HTTPClient
	username, password string
}

func (c *basicAuthHTTPClient) Do(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(c.username, c.password)
	return c.c.Do(req)
}

// HTTPClientWithBasicAuth returns an HTTP client that adds basic
// authentication to all outgoing requests. If c is nil, http.DefaultClient is
// used.
func HTTPClientWithBasicAuth(c HTTPClient, username, password string) HTTPClient {
	if c == nil {
		c = http.DefaultClient
	}
	return &basicAuthHTTPClient{c, username, password}
}

// Client provides access to a remote WebDAV filesystem.
type Client struct {
	ic *internal.Client
}

// NewClient creates a new WebDAV client.
//
// If the HTTPClient is nil, http.DefaultClient is used.
//
// To use HTTP basic authentication, HTTPClientWithBasicAuth can be used.
func NewClient(c HTTPClient, endpoint string) (*Client, error) {
	ic, err := internal.NewClient(c, endpoint)
	if err != nil {
		return nil, err
	}
	return &Client{ic}, nil
}

// FindCurrentUserPrincipal finds the current user's principal path.
func (c *Client) FindCurrentUserPrincipal(ctx context.Context) (string, error) {
	propfind := internal.NewPropNamePropFind(internal.CurrentUserPrincipalName)

	// TODO: consider retrying on the root URI "/" if this fails, as suggested
	// by the RFC?
	resp, err := c.ic.PropFindFlat(ctx, "", propfind)
	if err != nil {
		return "", err
	}

	var prop internal.CurrentUserPrincipal
	if err := resp.DecodeProp(&prop); err != nil {
		return "", err
	}
	if prop.Unauthenticated != nil {
		return "", fmt.Errorf("webdav: unauthenticated")
	}

	return prop.Href.Path, nil
}

var fileInfoPropFind = internal.NewPropNamePropFind(
	internal.ResourceTypeName,
	internal.GetContentLengthName,
	internal.GetLastModifiedName,
	internal.GetContentTypeName,
	internal.GetETagName,
)

func fileInfoFromResponse(resp *internal.Response) (*FileInfo, error) {
	path, err := resp.Path()
	if err != nil {
		return nil, err
	}

	fi := &FileInfo{Path: path}

	var resType internal.ResourceType
	if err := resp.DecodeProp(&resType); err != nil {
		return nil, err
	}

	if resType.Is(internal.CollectionName) {
		fi.IsDir = true
	} else {
		var getLen internal.GetContentLength
		if err := resp.DecodeProp(&getLen); err != nil {
			return nil, err
		}

		var getType internal.GetContentType
		if err := resp.DecodeProp(&getType); err != nil && !internal.IsNotFound(err) {
			return nil, err
		}

		var getETag internal.GetETag
		if err := resp.DecodeProp(&getETag); err != nil && !internal.IsNotFound(err) {
			return nil, err
		}

		fi.Size = getLen.Length
		fi.MIMEType = getType.Type
		fi.ETag = string(getETag.ETag)
	}

	var getMod internal.GetLastModified
	if err := resp.DecodeProp(&getMod); err != nil && !internal.IsNotFound(err) {
		return nil, err
	}
	fi.ModTime = time.Time(getMod.LastModified)

	return fi, nil
}

// Stat fetches a FileInfo for a single file.
func (c *Client) Stat(ctx context.Context, name string) (*FileInfo, error) {
	resp, err := c.ic.PropFindFlat(ctx, name, fileInfoPropFind)
	if err != nil {
		return nil, err
	}
	return fileInfoFromResponse(resp)
}

// Open fetches a file's contents.
func (c *Client) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	req, err := c.ic.NewRequest(http.MethodGet, name, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.ic.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// ReadDir lists files in a directory.
func (c *Client) ReadDir(ctx context.Context, name string, recursive bool) ([]FileInfo, error) {
	depth := internal.DepthOne
	if recursive {
		depth = internal.DepthInfinity
	}

	ms, err := c.ic.PropFind(ctx, name, depth, fileInfoPropFind)
	if err != nil {
		return nil, err
	}

	l := make([]FileInfo, 0, len(ms.Responses))
	for _, resp := range ms.Responses {
		fi, err := fileInfoFromResponse(&resp)
		if err != nil {
			return l, err
		}
		l = append(l, *fi)
	}

	return l, nil
}

type fileWriter struct {
	pw   *io.PipeWriter
	done <-chan error
}

func (fw *fileWriter) Write(b []byte) (int, error) {
	return fw.pw.Write(b)
}

func (fw *fileWriter) Close() error {
	if err := fw.pw.Close(); err != nil {
		return err
	}
	return <-fw.done
}

// Create writes a file's contents.
func (c *Client) Create(ctx context.Context, name string) (io.WriteCloser, error) {
	pr, pw := io.Pipe()

	req, err := c.ic.NewRequest(http.MethodPut, name, pr)
	if err != nil {
		pw.Close()
		return nil, err
	}

	done := make(chan error, 1)
	go func() {
		resp, err := c.ic.Do(req.WithContext(ctx))
		if err != nil {
			done <- err
			return
		}
		resp.Body.Close()
		done <- nil
	}()

	return &fileWriter{pw, done}, nil
}

// RemoveAll deletes a file. If the file is a directory, all of its descendants
// are recursively deleted as well.
func (c *Client) RemoveAll(ctx context.Context, name string) error {
	req, err := c.ic.NewRequest(http.MethodDelete, name, nil)
	if err != nil {
		return err
	}

	resp, err := c.ic.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// Mkdir creates a new directory.
func (c *Client) Mkdir(ctx context.Context, name string) error {
	req, err := c.ic.NewRequest("MKCOL", name, nil)
	if err != nil {
		return err
	}

	resp, err := c.ic.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// Copy copies a file.
//
// By default, if the file is a directory, all descendants are recursively
// copied as well.
func (c *Client) Copy(ctx context.Context, name, dest string, options *CopyOptions) error {
	if options == nil {
		options = new(CopyOptions)
	}

	req, err := c.ic.NewRequest("COPY", name, nil)
	if err != nil {
		return err
	}

	depth := internal.DepthInfinity
	if options.NoRecursive {
		depth = internal.DepthZero
	}

	req.Header.Set("Destination", c.ic.ResolveHref(dest).String())
	req.Header.Set("Overwrite", internal.FormatOverwrite(!options.NoOverwrite))
	req.Header.Set("Depth", depth.String())

	resp, err := c.ic.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// Move moves a file.
func (c *Client) Move(ctx context.Context, name, dest string, options *MoveOptions) error {
	if options == nil {
		options = new(MoveOptions)
	}

	req, err := c.ic.NewRequest("MOVE", name, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Destination", c.ic.ResolveHref(dest).String())
	req.Header.Set("Overwrite", internal.FormatOverwrite(!options.NoOverwrite))

	resp, err := c.ic.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
