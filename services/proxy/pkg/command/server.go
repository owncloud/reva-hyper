package command

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os/signal"
	"time"

	"github.com/cs3org/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/cs3org/reva/v2/pkg/store"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/justinas/alice"
	"github.com/urfave/cli/v2"
	microstore "go-micro.dev/v4/store"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"

	"github.com/owncloud/ocis/v2/ocis-pkg/config/configlog"
	"github.com/owncloud/ocis/v2/ocis-pkg/log"
	pkgmiddleware "github.com/owncloud/ocis/v2/ocis-pkg/middleware"
	"github.com/owncloud/ocis/v2/ocis-pkg/oidc"
	"github.com/owncloud/ocis/v2/ocis-pkg/registry"
	"github.com/owncloud/ocis/v2/ocis-pkg/runner"
	"github.com/owncloud/ocis/v2/ocis-pkg/service/grpc"
	"github.com/owncloud/ocis/v2/ocis-pkg/tracing"
	"github.com/owncloud/ocis/v2/ocis-pkg/version"
	policiessvc "github.com/owncloud/ocis/v2/protogen/gen/ocis/services/policies/v0"
	settingssvc "github.com/owncloud/ocis/v2/protogen/gen/ocis/services/settings/v0"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/config"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/config/parser"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/logging"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/metrics"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/middleware"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/proxy"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/router"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/server/debug"
	proxyHTTP "github.com/owncloud/ocis/v2/services/proxy/pkg/server/http"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/staticroutes"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/user/backend"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/userroles"
	ocisstore "github.com/owncloud/ocis/v2/services/store/pkg/store"
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
			userInfoCache := store.Create(
				store.Store(cfg.OIDC.UserinfoCache.Store),
				store.TTL(cfg.OIDC.UserinfoCache.TTL),
				store.Size(cfg.OIDC.UserinfoCache.Size),
				microstore.Nodes(cfg.OIDC.UserinfoCache.Nodes...),
				microstore.Database(cfg.OIDC.UserinfoCache.Database),
				microstore.Table(cfg.OIDC.UserinfoCache.Table),
				store.DisablePersistence(cfg.OIDC.UserinfoCache.DisablePersistence),
				store.Authentication(cfg.OIDC.UserinfoCache.AuthUsername, cfg.OIDC.UserinfoCache.AuthPassword),
			)

			var signingKeyStore microstore.Store
			if cfg.PreSignedURL.SigningKeys.Store == "ocisstoreservice" {
				signingKeyStore = ocisstore.NewStore(
					microstore.Nodes(cfg.PreSignedURL.SigningKeys.Nodes...),
					microstore.Database("proxy"),
					microstore.Table("signing-keys"),
				)
			} else {
				signingKeyStore = store.Create(
					store.Store(cfg.PreSignedURL.SigningKeys.Store),
					store.TTL(cfg.PreSignedURL.SigningKeys.TTL),
					microstore.Nodes(cfg.PreSignedURL.SigningKeys.Nodes...),
					microstore.Database("proxy"),
					microstore.Table("signing-keys"),
					store.Authentication(cfg.PreSignedURL.SigningKeys.AuthUsername, cfg.PreSignedURL.SigningKeys.AuthPassword),
				)
			}

			logger := logging.Configure(cfg.Service.Name, cfg.Log)
			traceProvider, err := tracing.GetServiceTraceProvider(cfg.Tracing, cfg.Service.Name)
			if err != nil {
				return err
			}
			cfg.GrpcClient, err = grpc.NewClient(
				append(
					grpc.GetClientOptions(cfg.GRPCClientTLS),
					grpc.WithTraceProvider(traceProvider))...)
			if err != nil {
				return err
			}

			oidcHTTPClient := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						MinVersion:         tls.VersionTLS12,
						InsecureSkipVerify: cfg.OIDC.Insecure, //nolint:gosec
					},
					DisableKeepAlives: true,
				},
				Timeout: time.Second * 10,
			}

			oidcClient := oidc.NewOIDCClient(
				oidc.WithAccessTokenVerifyMethod(cfg.OIDC.AccessTokenVerifyMethod),
				oidc.WithLogger(logger),
				oidc.WithHTTPClient(oidcHTTPClient),
				oidc.WithOidcIssuer(cfg.OIDC.Issuer),
				oidc.WithJWKSOptions(cfg.OIDC.JWKS),
			)

			var cancel context.CancelFunc
			ctx := cfg.Context
			if ctx == nil {
				ctx, cancel = signal.NotifyContext(context.Background(), runner.StopSignals...)
				defer cancel()
			}

			m := metrics.New()
			m.BuildInfo.WithLabelValues(version.GetString()).Set(1)

			rp, err := proxy.NewMultiHostReverseProxy(
				proxy.Logger(logger),
				proxy.Config(cfg),
			)

			lh := staticroutes.StaticRouteHandler{
				Prefix:         cfg.HTTP.Root,
				UserInfoCache:  userInfoCache,
				Logger:         logger,
				Config:         *cfg,
				OidcClient:     oidcClient,
				OidcHttpClient: oidcHTTPClient,
				Proxy:          rp,
			}
			if err != nil {
				return fmt.Errorf("failed to initialize reverse proxy: %w", err)
			}

			gr := runner.NewGroup()
			{
				middlewares := loadMiddlewares(ctx, logger, cfg, userInfoCache, signingKeyStore, traceProvider, *m)
				server, err := proxyHTTP.Server(
					proxyHTTP.Handler(lh.Handler()),
					proxyHTTP.Logger(logger),
					proxyHTTP.Context(ctx),
					proxyHTTP.Config(cfg),
					proxyHTTP.Metrics(metrics.New()),
					proxyHTTP.Middlewares(middlewares),
				)
				if err != nil {
					logger.Error().
						Err(err).
						Str("server", "http").
						Msg("Failed to initialize server")

					return err
				}

				gr.Add(runner.NewGoMicroHttpServerRunner("proxy_http", server))
			}

			{
				server, err := debug.Server(
					debug.Logger(logger),
					debug.Context(ctx),
					debug.Config(cfg),
				)
				if err != nil {
					logger.Error().Err(err).Str("server", "debug").Msg("Failed to initialize server")
					return err
				}

				gr.Add(runner.NewGolangHttpServerRunner("proxy_debug", server))
			}

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

func loadMiddlewares(ctx context.Context, logger log.Logger, cfg *config.Config, userInfoCache, signingKeyStore microstore.Store, traceProvider trace.TracerProvider, metrics metrics.Metrics) alice.Chain {
	rolesClient := settingssvc.NewRoleService("com.owncloud.api.settings", cfg.GrpcClient)
	policiesProviderClient := policiessvc.NewPoliciesProviderService("com.owncloud.api.policies", cfg.GrpcClient)
	gatewaySelector, err := pool.GatewaySelector(
		cfg.Reva.Address,
		append(
			cfg.Reva.GetRevaOptions(),
			pool.WithRegistry(registry.GetRegistry()),
			pool.WithTracerProvider(traceProvider),
		)...)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to get gateway selector")
	}
	if err != nil {
		logger.Fatal().Err(err).
			Msg("Failed to create token manager")
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
			backend.WithAutoProvisionClaims(cfg.AutoProvisionClaims),
		)
	default:
		logger.Fatal().Msgf("Invalid accounts backend type '%s'", cfg.AccountBackend)
	}

	var roleAssigner userroles.UserRoleAssigner
	switch cfg.RoleAssignment.Driver {
	case "default":
		roleAssigner = userroles.NewDefaultRoleAssigner(
			userroles.WithRoleService(rolesClient),
			userroles.WithLogger(logger),
		)
	case "oidc":
		roleAssigner = userroles.NewOIDCRoleAssigner(
			userroles.WithRoleService(rolesClient),
			userroles.WithLogger(logger),
			userroles.WithRolesClaim(cfg.RoleAssignment.OIDCRoleMapper.RoleClaim),
			userroles.WithRoleMapping(cfg.RoleAssignment.OIDCRoleMapper.RolesMap),
			userroles.WithRevaGatewaySelector(gatewaySelector),
			userroles.WithServiceAccount(cfg.ServiceAccount),
		)
	default:
		logger.Fatal().Msgf("Invalid role assignment driver '%s'", cfg.RoleAssignment.Driver)
	}

	oidcHTTPClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: cfg.OIDC.Insecure, //nolint:gosec
			},
			DisableKeepAlives: true,
		},
		Timeout: time.Second * 10,
	}

	var authenticators []middleware.Authenticator
	if cfg.EnableBasicAuth {
		logger.Warn().Msg("basic auth enabled, use only for testing or development")
		authenticators = append(authenticators, middleware.BasicAuthenticator{
			Logger:       logger,
			UserProvider: userProvider,
		})
	}

	authenticators = append(authenticators, middleware.PublicShareAuthenticator{
		Logger:              logger,
		RevaGatewaySelector: gatewaySelector,
	})
	authenticators = append(authenticators, middleware.NewOIDCAuthenticator(
		middleware.Logger(logger),
		middleware.UserInfoCache(userInfoCache),
		middleware.DefaultAccessTokenTTL(cfg.OIDC.UserinfoCache.TTL),
		middleware.HTTPClient(oidcHTTPClient),
		middleware.OIDCIss(cfg.OIDC.Issuer),
		middleware.OIDCClient(oidc.NewOIDCClient(
			oidc.WithAccessTokenVerifyMethod(cfg.OIDC.AccessTokenVerifyMethod),
			oidc.WithLogger(logger),
			oidc.WithHTTPClient(oidcHTTPClient),
			oidc.WithOidcIssuer(cfg.OIDC.Issuer),
			oidc.WithJWKSOptions(cfg.OIDC.JWKS),
		)),
		middleware.SkipUserInfo(cfg.OIDC.SkipUserInfo),
	))
	authenticators = append(authenticators, middleware.SignedURLAuthenticator{
		Logger:             logger,
		PreSignedURLConfig: cfg.PreSignedURL,
		UserProvider:       userProvider,
		UserRoleAssigner:   roleAssigner,
		Store:              signingKeyStore,
		Now:                time.Now,
	})

	cspConfig, err := middleware.LoadCSPConfig(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load CSP configuration.")
	}

	return alice.New(
		// first make sure we log all requests and redirect to https if necessary
		otelhttp.NewMiddleware("proxy",
			otelhttp.WithTracerProvider(traceProvider),
			otelhttp.WithSpanNameFormatter(func(name string, r *http.Request) string {
				return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
			}),
		),
		middleware.Tracer(traceProvider),
		pkgmiddleware.TraceContext,
		middleware.Instrumenter(metrics),
		chimiddleware.RealIP,
		chimiddleware.RequestID,
		middleware.AccessLog(logger),
		middleware.HTTPSRedirect,
		middleware.Security(cspConfig),
		router.Middleware(cfg.PolicySelector, cfg.Policies, logger),
		middleware.Authentication(
			authenticators,
			middleware.CredentialsByUserAgent(cfg.AuthMiddleware.CredentialsByUserAgent),
			middleware.Logger(logger),
			middleware.OIDCIss(cfg.OIDC.Issuer),
			middleware.EnableBasicAuth(cfg.EnableBasicAuth),
			middleware.TraceProvider(traceProvider),
		),
		middleware.AccountResolver(
			middleware.Logger(logger),
			middleware.UserProvider(userProvider),
			middleware.UserRoleAssigner(roleAssigner),
			middleware.SkipUserInfo(cfg.OIDC.SkipUserInfo),
			middleware.UserOIDCClaim(cfg.UserOIDCClaim),
			middleware.UserCS3Claim(cfg.UserCS3Claim),
			middleware.AutoprovisionAccounts(cfg.AutoprovisionAccounts),
		),
		middleware.SelectorCookie(
			middleware.Logger(logger),
			middleware.PolicySelectorConfig(*cfg.PolicySelector),
		),
		middleware.Policies(
			cfg.PoliciesMiddleware.Query,
			middleware.Logger(logger),
			middleware.WithRevaGatewaySelector(gatewaySelector),
			middleware.PoliciesProviderService(policiesProviderClient),
		),
		// finally, trigger home creation when a user logs in
		middleware.CreateHome(
			middleware.Logger(logger),
			middleware.WithRevaGatewaySelector(gatewaySelector),
			middleware.RoleQuotas(cfg.RoleQuotas),
		),
	)
}
