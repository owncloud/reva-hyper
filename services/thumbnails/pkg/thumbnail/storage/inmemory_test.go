package storage_test

import (
	"image"
	"testing"

	tAssert "github.com/stretchr/testify/assert"

	"github.com/owncloud/ocis/v2/services/thumbnails/pkg/thumbnail/storage"
)

func TestInMemory_BuildKey(t *testing.T) {
	tests := []struct {
		r    storage.Request
		want string
	}{
		{
			r: storage.Request{
				Checksum: "cs",
				Types:    []string{"png", "jpg"},
				Resolution: image.Rectangle{
					Min: image.Point{
						X: 1,
						Y: 2,
					},
					Max: image.Point{
						X: 3,
						Y: 4,
					},
				},
				Characteristic: "",
			},
			want: "cs+(1,2)-(3,4)+png,jpg",
		},
		{
			r: storage.Request{
				Checksum: "cs",
				Types:    []string{"png", "jpg"},
				Resolution: image.Rectangle{
					Min: image.Point{
						X: 1,
						Y: 2,
					},
					Max: image.Point{
						X: 3,
						Y: 4,
					},
				},
				Characteristic: "fill",
			},
			want: "cs+(1,2)-(3,4)+fill+png,jpg",
		},
	}

	s := storage.InMemory{}
	assert := tAssert.New(t)

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			assert.Equal(s.BuildKey(tt.r), tt.want)
		})
	}

}
