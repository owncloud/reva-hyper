package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/oklog/run"
	"github.com/owncloud/ocis/v2/ocis-pkg/config/configlog"
	"github.com/owncloud/ocis/v2/ocis-pkg/tracing"
	"github.com/owncloud/ocis/v2/services/web/pkg/config"
	"github.com/owncloud/ocis/v2/services/web/pkg/config/parser"
	"github.com/owncloud/ocis/v2/services/web/pkg/logging"
	"github.com/owncloud/ocis/v2/services/web/pkg/metrics"
	"github.com/owncloud/ocis/v2/services/web/pkg/server/debug"
	"github.com/owncloud/ocis/v2/services/web/pkg/server/http"
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

			// actually read the contents of the config file and override defaults
			if cfg.File != "" {
				contents, err := os.ReadFile(cfg.File)
				if err != nil {
					logger.Err(err).Msg("error opening config file")
					return err
				}
				if err = json.Unmarshal(contents, &cfg.Web.Config); err != nil {
					logger.Fatal().Err(err).Msg("error unmarshalling config file")
					return err
				}
			}

			var (
				gr          = run.Group{}
				ctx, cancel = context.WithCancel(c.Context)
				metrics     = metrics.New()
			)

			defer cancel()

			{
				server, err := http.Server(
					http.Logger(logger),
					http.Context(ctx),
					http.Namespace(cfg.HTTP.Namespace),
					http.Config(cfg),
					http.Metrics(metrics),
					http.TraceProvider(traceProvider),
				)
				if err != nil {
					logger.Info().
						Err(err).
						Str("transport", "http").
						Msg("Failed to initialize server")

					return err
				}

				gr.Add(func() error {
					err := server.Run()
					if err != nil {
						logger.Error().
							Err(err).
							Str("transport", "http").
							Msg("Failed to start server")
					}
					return err
				}, func(err error) {
					logger.Info().
						Str("transport", "http").
						Msg("Shutting down server")

					cancel()
				})
			}

			{
				server, err := debug.Server(
					debug.Logger(logger),
					debug.Context(ctx),
					debug.Config(cfg),
				)
				if err != nil {
					logger.Info().Err(err).Str("transport", "debug").Msg("Failed to initialize server")
					return err
				}

				gr.Add(server.ListenAndServe, func(_ error) {
					_ = server.Shutdown(ctx)
					cancel()
				})
			}

			return gr.Run()
		},
	}
}
