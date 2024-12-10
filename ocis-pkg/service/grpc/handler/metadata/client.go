package metadata

import (
	"context"

	"go-micro.dev/v4/client"
	"go-micro.dev/v4/metadata"
)

type clientWrapper struct {
	client.Client
	mdata metadata.Metadata
}

func NewClientWrapper(mdata map[string]string) client.Wrapper {
	meta := metadata.Metadata(mdata)
	return func(c client.Client) client.Client {
		w := &clientWrapper{
			Client: c,
			mdata:  meta,
		}
		return w
	}
}

func (w *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	newCtx := metadata.MergeContext(ctx, w.mdata, true)
	return w.Client.Call(newCtx, req, rsp, opts...)
}

func (w *clientWrapper) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	newCtx := metadata.MergeContext(ctx, w.mdata, true)
	return w.Client.Stream(newCtx, req, opts...)
}

func (w *clientWrapper) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	newCtx := metadata.MergeContext(ctx, w.mdata, true)
	return w.Client.Publish(newCtx, p, opts...)
}
