package thumbnail

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/owncloud/ocis/v2/ocis-pkg/log"
	"github.com/owncloud/ocis/v2/services/thumbnails/pkg/thumbnail/storage"
)

type NoOpManager struct {
	storage.Storage
}

func (m NoOpManager) BuildKey(r storage.Request) string {
	return ""
}

func (m NoOpManager) Set(username, key string, thumbnail []byte) error {
	return nil
}

func BenchmarkGet(b *testing.B) {

	sut := NewSimpleManager(
		Resolutions{},
		NoOpManager{},
		log.NewLogger(),
	)

	res, _ := ParseResolution("32x32")
	req := Request{
		Resolution: res,
		Checksum:   "1872ade88f3013edeb33decd74a4f947",
	}
	cwd, _ := os.Getwd()
	p := filepath.Join(cwd, "../../testdata/oc.png")
	f, _ := os.Open(p)
	defer f.Close()
	img, ext, _ := image.Decode(f)
	req.Encoder, _ = EncoderForType(ext)
	for i := 0; i < b.N; i++ {
		_, _ = sut.Generate(req, img)
	}
}

func TestPrepareRequest(t *testing.T) {
	type args struct {
		width    int
		height   int
		tType    string
		checksum string
	}
	tests := []struct {
		name    string
		args    args
		want    Request
		wantErr bool
	}{
		{
			name: "Test successful prepare the request for jpg",
			args: args{
				width:    32,
				height:   32,
				tType:    "jpg",
				checksum: "1872ade88f3013edeb33decd74a4f947",
			},
			want: Request{
				Resolution: image.Rect(0, 0, 32, 32),
				Encoder:    JpegEncoder{},
				Generator:  SimpleGenerator{},
				Checksum:   "1872ade88f3013edeb33decd74a4f947",
			},
		},
		{
			name: "Test successful prepare the request for png",
			args: args{
				width:    32,
				height:   32,
				tType:    "png",
				checksum: "1872ade88f3013edeb33decd74a4f947",
			},
			want: Request{
				Resolution: image.Rect(0, 0, 32, 32),
				Encoder:    PngEncoder{},
				Generator:  SimpleGenerator{},
				Checksum:   "1872ade88f3013edeb33decd74a4f947",
			},
		},
		{
			name: "Test successful prepare the request for gif",
			args: args{
				width:    32,
				height:   32,
				tType:    "gif",
				checksum: "1872ade88f3013edeb33decd74a4f947",
			},
			want: Request{
				Resolution: image.Rect(0, 0, 32, 32),
				Encoder:    GifEncoder{},
				Generator:  GifGenerator{},
				Checksum:   "1872ade88f3013edeb33decd74a4f947",
			},
		},
		{
			name: "Test error when prepare the request for bmp",
			args: args{
				width:    32,
				height:   32,
				tType:    "bmp",
				checksum: "1872ade88f3013edeb33decd74a4f947",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PrepareRequest(tt.args.width, tt.args.height, tt.args.tType, tt.args.checksum, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("PrepareRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// func's are not reflactable, ignore
			if diff := cmp.Diff(tt.want, got, cmpopts.IgnoreFields(Request{}, "Processor")); diff != "" {
				t.Errorf("PrepareRequest(): %v", diff)
			}
		})
	}
}
