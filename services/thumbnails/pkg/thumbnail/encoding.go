package thumbnail

import (
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
)

const (
	typePng  = "png"
	typeJpg  = "jpg"
	typeJpeg = "jpeg"
	typeGif  = "gif"
)

var (
	// ErrInvalidType represents the error when a type can't be encoded.
	ErrInvalidType = errors.New("can't encode this type")
	// ErrNoEncoderForType represents the error when an encoder couldn't be found for a type.
	ErrNoEncoderForType = errors.New("no encoder for this type found")
)

// Encoder encodes the thumbnail to a specific format.
type Encoder interface {
	// Encode encodes the image to a format.
	Encode(io.Writer, interface{}) error
	// Types returns the formats suffixes.
	Types() []string
	// MimeType returns the mimetype used by the encoder.
	MimeType() string
}

// PngEncoder encodes to png
type PngEncoder struct{}

// Encode encodes to png format
func (e PngEncoder) Encode(w io.Writer, img interface{}) error {
	m, ok := img.(image.Image)
	if !ok {
		return ErrInvalidType
	}
	return png.Encode(w, m)
}

// Types returns the png suffix
func (e PngEncoder) Types() []string {
	return []string{typePng}
}

// MimeType returns the mimetype for png files.
func (e PngEncoder) MimeType() string {
	return "image/png"
}

// JpegEncoder encodes to jpg.
type JpegEncoder struct{}

// Encode encodes to jpg
func (e JpegEncoder) Encode(w io.Writer, img interface{}) error {
	m, ok := img.(image.Image)
	if !ok {
		return ErrInvalidType
	}
	return jpeg.Encode(w, m, nil)
}

// Types returns the jpg suffixes.
func (e JpegEncoder) Types() []string {
	return []string{typeJpeg, typeJpg}
}

// MimeType returns the mimetype for jpg files.
func (e JpegEncoder) MimeType() string {
	return "image/jpeg"
}

type GifEncoder struct{}

// Encode encodes the image to a gif format
func (e GifEncoder) Encode(w io.Writer, img interface{}) error {
	g, ok := img.(*gif.GIF)
	if !ok {
		return ErrInvalidType
	}
	return gif.EncodeAll(w, g)
}

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
	case typePng:
		return PngEncoder{}, nil
	case typeJpg, typeJpeg:
		return JpegEncoder{}, nil
	case typeGif:
		return GifEncoder{}, nil
	default:
		return nil, ErrNoEncoderForType
	}
}

// GetExtForMime return the supported extension by mime
func GetExtForMime(fileType string) string {
	ext := strings.TrimPrefix(strings.TrimSpace(strings.ToLower(fileType)), "image/")
	switch ext {
	case typeJpg, typeJpeg, typePng, typeGif:
		return ext
	default:
		return ""
	}
}
