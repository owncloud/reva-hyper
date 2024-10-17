package command

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/cs3org/reva/v2/cmd/revad/runtime"
	"github.com/cs3org/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/gofrs/uuid"
	"github.com/oklog/run"
	"github.com/owncloud/ocis/v2/ocis-pkg/config/configlog"
	"github.com/owncloud/ocis/v2/ocis-pkg/registry"
	"github.com/owncloud/ocis/v2/ocis-pkg/sync"
	"github.com/owncloud/ocis/v2/ocis-pkg/tracing"
	"github.com/owncloud/ocis/v2/ocis-pkg/version"
	"github.com/owncloud/ocis/v2/services/storage-users/pkg/config"
	"github.com/owncloud/ocis/v2/services/storage-users/pkg/config/parser"
	"github.com/owncloud/ocis/v2/services/storage-users/pkg/event"
	"github.com/owncloud/ocis/v2/services/storage-users/pkg/logging"
	"github.com/owncloud/ocis/v2/services/storage-users/pkg/revaconfig"
	"github.com/owncloud/ocis/v2/services/storage-users/pkg/server/debug"
	"github.com/urfave/cli/v2"
)

// Server is the entry point for the server command.
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
			gr := run.Group{}
			ctx, cancel := context.WithCancel(c.Context)

			defer cancel()

			gr.Add(func() error {
				pidFile := path.Join(os.TempDir(), "revad-"+cfg.Service.Name+"-"+uuid.Must(uuid.NewV4()).String()+".pid")
				rCfg := revaconfig.StorageUsersConfigFromStruct(cfg)
				reg := registry.GetRegistry()

				runtime.RunWithOptions(rCfg, pidFile,
					runtime.WithLogger(&logger.Logger),
					runtime.WithRegistry(reg),
					runtime.WithTraceProvider(traceProvider),
				)

				return nil
			}, func(err error) {
				logger.Error().
					Err(err).
					Str("server", cfg.Service.Name).
					Msg("Shutting down server")

				cancel()
			})

			debugServer, err := debug.Server(
				debug.Logger(logger),
				debug.Context(ctx),
				debug.Config(cfg),
			)
			if err != nil {
				logger.Info().Err(err).Str("server", "debug").Msg("Failed to initialize server")
				return err
			}

			gr.Add(debugServer.ListenAndServe, func(err error) {
				logger.Error().Err(err).Str("server", cfg.Service.Name).
					Msg("Shutting down debug server")
				if err := debugServer.Shutdown(context.Background()); err != nil {
					logger.Error().
						Err(err).
						Str("server", cfg.Service.Name).
						Msg("Error during debug server shutdown")
				}

				cancel()
			})

			if !cfg.Supervised {
				sync.Trap(&gr, cancel)
			}

			grpcSvc := registry.BuildGRPCService(cfg.GRPC.Namespace+"."+cfg.Service.Name, cfg.GRPC.Addr, version.GetString())
			if err := registry.RegisterService(ctx, grpcSvc, logger); err != nil {
				logger.Fatal().Err(err).Msg("failed to register the grpc service")
			}

			{
				stream, err := event.NewStream(cfg)
				if err != nil {
					logger.Fatal().Err(err).Msg("can't connect to nats")
				}

				selector, err := pool.GatewaySelector(cfg.Reva.Address, pool.WithRegistry(registry.GetRegistry()), pool.WithTracerProvider(traceProvider))
				if err != nil {
					return err
				}

				eventSVC, err := event.NewService(ctx, selector, stream, logger, *cfg)
				if err != nil {
					logger.Fatal().Err(err).Msg("can't create event handler")
				}

				gr.Add(eventSVC.Run, func(_ error) {
					logger.Error().
						Err(err).
						Str("server", cfg.Service.Name).
						Msg("Shutting down event handler")

					cancel()
				})
			}

			return gr.Run()
		},
	}
}
