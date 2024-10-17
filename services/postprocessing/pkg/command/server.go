package command

import (
	"context"
	"fmt"
	"os"

	"github.com/cs3org/reva/v2/pkg/events/stream"
	"github.com/cs3org/reva/v2/pkg/store"
	"github.com/oklog/run"
	"github.com/owncloud/ocis/v2/ocis-pkg/handlers"
	"github.com/owncloud/ocis/v2/ocis-pkg/service/debug"
	"github.com/owncloud/ocis/v2/ocis-pkg/tracing"
	"github.com/owncloud/ocis/v2/ocis-pkg/version"
	"github.com/owncloud/ocis/v2/services/postprocessing/pkg/config"
	"github.com/owncloud/ocis/v2/services/postprocessing/pkg/config/parser"
	"github.com/owncloud/ocis/v2/services/postprocessing/pkg/logging"
	"github.com/owncloud/ocis/v2/services/postprocessing/pkg/service"
	"github.com/urfave/cli/v2"
	microstore "go-micro.dev/v4/store"
)

// Server is the entrypoint for the server command.
func Server(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "server",
		Usage:    fmt.Sprintf("start %s service without runtime (unsupervised mode)", cfg.Service.Name),
		Category: "server",
		Before: func(c *cli.Context) error {
			err := parser.ParseConfig(cfg)
			if err != nil {
				fmt.Printf("%v", err)
				os.Exit(1)
			}
			return err
		},
		Action: func(c *cli.Context) error {
			var (
				gr          = run.Group{}
				logger      = logging.Configure(cfg.Service.Name, cfg.Log)
				ctx, cancel = context.WithCancel(c.Context)
			)
			defer cancel()

			traceProvider, err := tracing.GetServiceTraceProvider(cfg.Tracing, cfg.Service.Name)
			if err != nil {
				return err
			}

			{
				bus, err := stream.NatsFromConfig(cfg.Service.Name, false, stream.NatsConfig(cfg.Postprocessing.Events))
				if err != nil {
					return err
				}

				st := store.Create(
					store.Store(cfg.Store.Store),
					store.TTL(cfg.Store.TTL),
					store.Size(cfg.Store.Size),
					microstore.Nodes(cfg.Store.Nodes...),
					microstore.Database(cfg.Store.Database),
					microstore.Table(cfg.Store.Table),
					store.Authentication(cfg.Store.AuthUsername, cfg.Store.AuthPassword),
				)

				svc, err := service.NewPostprocessingService(ctx, bus, logger, st, traceProvider, cfg.Postprocessing)
				if err != nil {
					return err
				}
				gr.Add(func() error {
					err := make(chan error)
					select {
					case <-ctx.Done():
						return nil

					case err <- svc.Run():
						return <-err
					}
				}, func(err error) {
					logger.Error().
						Err(err).
						Msg("Shutting down server")
					cancel()
				})
			}

			{
				server := debug.NewService(
					debug.Logger(logger),
					debug.Name(cfg.Service.Name),
					debug.Version(version.GetString()),
					debug.Address(cfg.Debug.Addr),
					debug.Token(cfg.Debug.Token),
					debug.Pprof(cfg.Debug.Pprof),
					debug.Zpages(cfg.Debug.Zpages),
					debug.Health(handlers.Health),
					debug.Ready(handlers.Ready),
				)

				gr.Add(server.ListenAndServe, func(_ error) {
					_ = server.Shutdown(ctx)
					cancel()
				})
			}
			return gr.Run()
		},
	}
}
