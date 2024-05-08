package net

import (
	"net/url"
)

// EncodePath encodes the path of an url.
//
// slashes (/) are treated as path-separators.
func EncodePath(path string) string {
	return (&url.URL{Path: path}).EscapedPath()
}
