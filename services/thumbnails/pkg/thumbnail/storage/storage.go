package storage

import (
	"image"
)

// Request combines different attributes needed for storage operations.
type Request struct {
	// The checksum of the source file
	// Will be used to determine if a thumbnail exists
	Checksum string
	// Types provided by the encoder.
	// Contains the mimetypes of the thumbnail.
	// In case of jpg/jpeg it will contain both.
	Types []string
	// The resolution of the thumbnail
	Resolution image.Rectangle
	// Characteristic defines the different image characteristics,
	// for example, if its scaled up to fit in the bounding box or not,
	// is it a chroma version of the image, and so on...
	// the main propose for this is to be able to differentiate between images which have
	// the same resolution but different characteristics.
	Characteristic string
}

// Storage defines the interface for a thumbnail store.
type Storage interface {
	Stat(string) bool
	Get(string) ([]byte, error)
	Put(string, []byte) error
	BuildKey(Request) string
}
