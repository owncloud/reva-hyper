package command

import (
	"context"
	"fmt"
	"net"
	"os/signal"

	"github.com/owncloud/ocis/v2/ocis-pkg/config/configlog"
	"github.com/owncloud/ocis/v2/ocis-pkg/runner"
	"github.com/owncloud/ocis/v2/ocis-pkg/tracing"
	"github.com/owncloud/ocis/v2/services/collaboration/pkg/config"
	"github.com/owncloud/ocis/v2/services/collaboration/pkg/config/parser"
	"github.com/owncloud/ocis/v2/services/collaboration/pkg/connector"
	"github.com/owncloud/ocis/v2/services/collaboration/pkg/helpers"
	"github.com/owncloud/ocis/v2/services/collaboration/pkg/logging"
	"github.com/owncloud/ocis/v2/services/collaboration/pkg/server/debug"
	"github.com/owncloud/ocis/v2/services/collaboration/pkg/server/grpc"
	"github.com/owncloud/ocis/v2/services/collaboration/pkg/server/http"
	"github.com/urfave/cli/v2"
)

// Server is the entrypoint for the server command.
func Server(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "server",
		Usage:    fmt.Sprintf("start the %s service without runtime (unsupervised mode)", cfg.Service.Name),
		Category: "server",
		Before: func(c *cli.Context) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		Action: func(c *cli.Context) error {
			logger := logging.Configure(cfg.Service.Name, cfg.Log)
			traceProvider, err := tracing.GetServiceTraceProvider(cfg.Tracing, cfg.Service.Name)
			if err != nil {
				return err
			}

			var cancel context.CancelFunc
			ctx := cfg.Context
			if ctx == nil {
				ctx, cancel = signal.NotifyContext(context.Background(), runner.StopSignals...)
				defer cancel()
			}

			// prepare components
			if err := helpers.RegisterOcisService(ctx, cfg, logger); err != nil {
				return err
			}

			gwc, err := helpers.GetCS3apiClient(cfg, false)
			if err != nil {
				return err
			}

			appUrls, err := helpers.GetAppURLs(cfg, logger)
			if err != nil {
				return err
			}

			if err := helpers.RegisterAppProvider(ctx, cfg, logger, gwc, appUrls); err != nil {
				return err
			}

			gr := runner.NewGroup()

			// start GRPC server
			grpcServer, teardown, err := grpc.Server(
				grpc.AppURLs(appUrls),
				grpc.Config(cfg),
				grpc.Logger(logger),
			)
			defer teardown()
			if err != nil {
				logger.Info().
					Err(err).
					Str("transport", "grpc").
					Msg("Failed to initialize server")
				return err
			}

			l, err := net.Listen("tcp", cfg.GRPC.Addr)
			if err != nil {
				return err
			}
			gr.Add(runner.NewGolangGrpcServerRunner("collaboration_grpc", grpcServer, l))

			// start debug server
			debugServer, err := debug.Server(
				debug.Logger(logger),
				debug.Context(ctx),
				debug.Config(cfg),
			)
			if err != nil {
				logger.Info().Err(err).Str("transport", "debug").Msg("Failed to initialize server")
				return err
			}
			gr.Add(runner.NewGolangHttpServerRunner("collaboration_debug", debugServer))

			// start HTTP server
			httpServer, err := http.Server(
				http.Adapter(connector.NewHttpAdapter(gwc, cfg)),
				http.Logger(logger),
				http.Config(cfg),
				http.Context(ctx),
				http.TracerProvider(traceProvider),
			)
			gr.Add(runner.NewGoMicroHttpServerRunner("collaboration_http", httpServer))

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
