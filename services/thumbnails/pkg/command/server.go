package command

import (
	"context"
	"fmt"
	"os/signal"

	"github.com/owncloud/ocis/v2/ocis-pkg/config/configlog"
	"github.com/owncloud/ocis/v2/ocis-pkg/runner"
	ogrpc "github.com/owncloud/ocis/v2/ocis-pkg/service/grpc"
	"github.com/owncloud/ocis/v2/ocis-pkg/tracing"
	"github.com/owncloud/ocis/v2/ocis-pkg/version"
	"github.com/owncloud/ocis/v2/services/thumbnails/pkg/config"
	"github.com/owncloud/ocis/v2/services/thumbnails/pkg/config/parser"
	"github.com/owncloud/ocis/v2/services/thumbnails/pkg/logging"
	"github.com/owncloud/ocis/v2/services/thumbnails/pkg/metrics"
	"github.com/owncloud/ocis/v2/services/thumbnails/pkg/server/debug"
	"github.com/owncloud/ocis/v2/services/thumbnails/pkg/server/grpc"
	"github.com/owncloud/ocis/v2/services/thumbnails/pkg/server/http"
	"github.com/urfave/cli/v2"
)

// Server is the entrypoint for the server command.
func Server(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "server",
		Usage:    fmt.Sprintf("start the %s service without runtime (unsupervised mode)", cfg.Service.Name),
		Category: "server",
		Before: func(_ *cli.Context) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		Action: func(_ *cli.Context) error {
			logger := logging.Configure(cfg.Service.Name, cfg.Log)

			traceProvider, err := tracing.GetServiceTraceProvider(cfg.Tracing, cfg.Service.Name)
			if err != nil {
				return err
			}
			cfg.GrpcClient, err = ogrpc.NewClient(ogrpc.GetClientOptions(cfg.GRPCClientTLS)...)
			if err != nil {
				return err
			}

			var cancel context.CancelFunc
			ctx := cfg.Context
			if ctx == nil {
				ctx, cancel = signal.NotifyContext(context.Background(), runner.StopSignals...)
				defer cancel()
			}

			m := metrics.New()
			m.BuildInfo.WithLabelValues(version.GetString()).Set(1)

			gr := runner.NewGroup()

			service := grpc.NewService(
				grpc.Logger(logger),
				grpc.Context(ctx),
				grpc.Config(cfg),
				grpc.Name(cfg.Service.Name),
				grpc.Namespace(cfg.GRPC.Namespace),
				grpc.Address(cfg.GRPC.Addr),
				grpc.Metrics(m),
				grpc.TraceProvider(traceProvider),
			)
			gr.Add(runner.NewGoMicroGrpcServerRunner("thumbnails_grpc", service))

			server, err := debug.Server(
				debug.Logger(logger),
				debug.Config(cfg),
			)
			if err != nil {
				logger.Info().Err(err).Str("transport", "debug").Msg("Failed to initialize server")
				return err
			}
			gr.Add(runner.NewGolangHttpServerRunner("thumbnails_debug", server))

			httpServer, err := http.Server(
				http.Logger(logger),
				http.Context(ctx),
				http.Config(cfg),
				http.Metrics(m),
				http.Namespace(cfg.HTTP.Namespace),
				http.TraceProvider(traceProvider),
				http.MaxConcurrentRequests(cfg.HTTP.MaxConcurrentRequests),
			)
			if err != nil {
				logger.Info().
					Err(err).
					Str("transport", "http").
					Msg("Failed to initialize server")

				return err
			}
			gr.Add(runner.NewGoMicroHttpServerRunner("thumbnails_http", httpServer))

			grResults := gr.Run(ctx)

			// return the first non-nil error found in the results
			for _, grResult := range grResults {
				if grResult.RunnerError != nil {
					return grResult.RunnerError
				}
			}
			return nil
		},
	}
}
