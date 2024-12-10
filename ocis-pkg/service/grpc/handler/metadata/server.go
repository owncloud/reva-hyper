package metadata

import (
	"context"
	"sort"

	"github.com/owncloud/ocis/v2/ocis-pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	micrometa "go-micro.dev/v4/metadata"
	"go-micro.dev/v4/server"
)

func NewMetadataLogHandler(l *log.Logger) func(fn server.HandlerFunc) server.HandlerFunc {
	return func(fn server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			if mdata, ok := micrometa.FromContext(ctx); ok {
				keys := make([]string, 0, len(mdata))
				for k, _ := range mdata {
					keys = append(keys, k)
				}
				sort.Strings(keys)

				logContext := l.With()
				for _, sortedKey := range keys {
					value, _ := mdata.Get(sortedKey)
					logContext = logContext.Str(sortedKey, value)
				}

				mdataLogger := logContext.Str("target", req.Service()).
					Str("method", req.Method()).
					Str("endpoint", req.Endpoint()).
					Bool("isStream", req.Stream()).
					Logger()
				ctx = mdataLogger.WithContext(ctx)
			}
			return fn(ctx, req, rsp)
		}
	}
}

func NewUnaryInterceptor(l *log.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if mdata, ok := metadata.FromIncomingContext(ctx); ok {
			keys := make([]string, 0, len(mdata))
			for k, _ := range mdata {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			logContext := l.With()
			for _, sortedKey := range keys {
				values := mdata.Get(sortedKey)
				logContext = logContext.Strs(sortedKey, values)
			}
			mdataLogger := logContext.Logger()
			ctx = mdataLogger.WithContext(ctx)
		}
		return handler(ctx, req)
	}
}

type serverStreamWrapper struct {
	grpc.ServerStream
	ctx context.Context
}

func newServerStreamWrapper(ss grpc.ServerStream, ctx context.Context) grpc.ServerStream {
	return &serverStreamWrapper{ss, ctx}
}

func (w *serverStreamWrapper) Context() context.Context {
	return w.ctx
}

func NewStreamInterceptor(l *log.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		if mdata, ok := metadata.FromIncomingContext(ctx); ok {
			keys := make([]string, 0, len(mdata))
			for k, _ := range mdata {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			logContext := l.With()
			for _, sortedKey := range keys {
				values := mdata.Get(sortedKey)
				logContext = logContext.Strs(sortedKey, values)
			}
			mdataLogger := logContext.Logger()
			ctx = mdataLogger.WithContext(ctx)
		}
		return handler(srv, newServerStreamWrapper(ss, ctx))
	}
}
