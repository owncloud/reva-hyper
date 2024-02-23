package command

import (
	"fmt"
	"github.com/cs3org/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/cs3org/reva/v2/pkg/store"
	"github.com/owncloud/ocis/v2/ocis-pkg/config/configlog"
	"github.com/owncloud/ocis/v2/ocis-pkg/registry"
	"github.com/owncloud/ocis/v2/ocis-pkg/tracing"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/config"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/config/parser"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/logging"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/middleware"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/user/backend"
	"github.com/urfave/cli/v2"
	microstore "go-micro.dev/v4/store"
)

// ClientAPIKey is the entrypoint for the client api key commands.
func ClientAPIKey(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "generate-client-api-key",
		Usage:    "generate client API key",
		Category: "maintenance",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "claim",
				Value:    "string",
				Usage:    "claim to search for the user: userid, username or email",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "value",
				Value:    "string",
				Usage:    "value to search for the user",
				Required: true,
			},
		},
		Before: func(_ *cli.Context) error {
			return configlog.ReturnError(parser.ParseConfig(cfg))
		},
		Action: func(c *cli.Context) error {
			logger := logging.Configure(cfg.Service.Name, cfg.Log)

			s := store.Create(
				store.Store(cfg.ClientAPIKeyStore.Store),
				store.TTL(cfg.ClientAPIKeyStore.TTL),
				microstore.Nodes(cfg.ClientAPIKeyStore.Nodes...),
				microstore.Database("proxy"),
				microstore.Table("client_api_keys"),
				store.Authentication(cfg.ClientAPIKeyStore.AuthUsername, cfg.ClientAPIKeyStore.AuthPassword),
			)
			traceProvider, err := tracing.GetServiceTraceProvider(cfg.Tracing, cfg.Service.Name)
			if err != nil {
				return err
			}

			gatewaySelector, err := pool.GatewaySelector(
				cfg.Reva.Address,
				append(
					cfg.Reva.GetRevaOptions(),
					pool.WithRegistry(registry.GetRegistry()),
					pool.WithTracerProvider(traceProvider),
				)...)
			if err != nil {
				return err
			}

			var userProvider backend.UserBackend
			switch cfg.AccountBackend {
			case "cs3":
				userProvider = backend.NewCS3UserBackend(
					backend.WithLogger(logger),
					backend.WithRevaGatewaySelector(gatewaySelector),
					backend.WithMachineAuthAPIKey(cfg.MachineAuthAPIKey),
					backend.WithOIDCissuer(cfg.OIDC.Issuer),
					backend.WithServiceAccount(cfg.ServiceAccount),
				)
			default:
				logger.Fatal().Msgf("Invalid accounts backend type '%s'", cfg.AccountBackend)
			}

			claim := c.String("claim")
			value := c.String("value")
			user, _, err := userProvider.GetUserByClaims(c.Context, claim, value)
			if err != nil {
				return err
			}

			auth := middleware.ClientAPIKeyAuthenticator{
				Logger:       logger,
				UserProvider: userProvider,
				SigningKey:   cfg.MachineAuthAPIKey,
				Store:        s,
			}
			clientAPIKey, clientAPISecret, err := auth.CreateClientAPIKey()
			if err != nil {
				return err
			}
			err = auth.SaveKey(user.Id.OpaqueId, clientAPIKey)
			if err != nil {
				return err
			}

			fmt.Printf("Client API key created for %s", user.Username)
			fmt.Println()
			fmt.Printf(" id    : %s", clientAPIKey)
			fmt.Println()
			fmt.Printf(" secret: %s", clientAPISecret)
			fmt.Println()

			return nil
		},
	}
}
