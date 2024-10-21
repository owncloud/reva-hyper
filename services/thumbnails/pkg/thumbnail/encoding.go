package thumbnail

import (
	"image/gif"
	"io"
	"strings"

	"github.com/owncloud/ocis/v2/services/thumbnails/pkg/errors"
)

const (
	typePng  = "png"
	typeJpg  = "jpg"
	typeJpeg = "jpeg"
	typeGif  = "gif"
	typeGgs  = "ggs"
	typeHeic = "heic"
	typeWebp = "webp"
)

// Encoder encodes the thumbnail to a specific format.
type Encoder interface {
	// Encode encodes the image to a format.
	Encode(w io.Writer, img interface{}) error
	// Types returns the formats suffixes.
	Types() []string
	// MimeType returns the mimetype used by the encoder.
	MimeType() string
}

// GifEncoder encodes to gif
type GifEncoder struct{}

// Encode encodes the image to a gif format
func (e GifEncoder) Encode(w io.Writer, img interface{}) error {
	g, ok := img.(*gif.GIF)
	if !ok {
		return errors.ErrInvalidType
	}
	return gif.EncodeAll(w, g)
}

// Types returns the supported types of the GifEncoder
func (e GifEncoder) Types() []string {
	return []string{typeGif}
}

// MimeType returns the mimetype used by the encoder.
func (e GifEncoder) MimeType() string {
	return "image/gif"
}

// EncoderForType returns the encoder for a given file type
// or nil if the type is not supported.
func EncoderForType(fileType string) (Encoder, error) {
	switch strings.ToLower(fileType) {
	case typePng, typeGgs:
		return PngEncoder{}, nil
	case typeJpg, typeJpeg, typeHeic, typeWebp:
		return JpegEncoder{}, nil
	case typeGif:
		return GifEncoder{}, nil
	default:
		return nil, errors.ErrNoEncoderForType
	}
}

// GetExtForMime return the supported extension by mime
func GetExtForMime(fileType string) string {
	ext := strings.TrimPrefix(strings.TrimSpace(strings.ToLower(fileType)), "image/")
	switch ext {
	case typeJpg, typeJpeg, typePng, typeGif, typeHeic, typeWebp:
		return ext
	case "application/vnd.geogebra.slides":
		return typeGgs
	default:
		return ""
	}
}
