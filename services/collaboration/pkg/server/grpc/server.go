package grpc

import (
	appproviderv1beta1 "github.com/cs3org/go-cs3apis/cs3/app/provider/v1beta1"
	"github.com/owncloud/ocis/v2/ocis-pkg/service/grpc/handler/metadata"
	"github.com/owncloud/ocis/v2/ocis-pkg/tracing"
	svc "github.com/owncloud/ocis/v2/services/collaboration/pkg/service/grpc/v0"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// Server initializes a new grpc service ready to run
// THIS SERVICE IS REGISTERED AGAINST REVA, NOT GO-MICRO
func Server(opts ...Option) (*grpc.Server, func(), error) {
	options := newOptions(opts...)

	grpcOpts := []grpc.ServerOption{
		grpc.StatsHandler(
			otelgrpc.NewServerHandler(
				otelgrpc.WithTracerProvider(options.TraceProvider),
				otelgrpc.WithPropagators(tracing.GetPropagator()),
			),
		),
		grpc.ChainUnaryInterceptor(
			metadata.NewUnaryInterceptor(&options.Logger),
		),
		grpc.ChainStreamInterceptor(
			metadata.NewStreamInterceptor(&options.Logger),
		),
	}
	grpcServer := grpc.NewServer(grpcOpts...)

	handle, teardown, err := svc.NewHandler(
		svc.Config(options.Config),
		svc.Logger(options.Logger),
		svc.AppURLs(options.AppURLs),
		svc.Store(options.Store),
	)
	if err != nil {
		options.Logger.Error().
			Err(err).
			Msg("Error initializing collaboration service")
		return grpcServer, teardown, err
	}

	// register the app provider interface / OpenInApp call
	appproviderv1beta1.RegisterProviderAPIServer(grpcServer, handle)

	return grpcServer, teardown, nil
}
