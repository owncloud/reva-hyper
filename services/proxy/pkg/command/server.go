package command

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/justinas/alice"
	"github.com/oklog/run"

	"github.com/cs3org/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/cs3org/reva/v2/pkg/store"
	"github.com/owncloud/ocis/v2/ocis-pkg/config/configlog"
	"github.com/owncloud/ocis/v2/ocis-pkg/log"
	pkgmiddleware "github.com/owncloud/ocis/v2/ocis-pkg/middleware"
	"github.com/owncloud/ocis/v2/ocis-pkg/oidc"
	"github.com/owncloud/ocis/v2/ocis-pkg/registry"
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
	"github.com/owncloud/ocis/v2/services/proxy/pkg/user/backend"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/userroles"
	ocisstore "github.com/owncloud/ocis/v2/services/store/pkg/store"
	"github.com/urfave/cli/v2"
	"go-micro.dev/v4/selector"
	microstore "go-micro.dev/v4/store"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
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

			m := metrics.New()

			gr := run.Group{}
			ctx, cancel := context.WithCancel(c.Context)

			defer cancel()

			m.BuildInfo.WithLabelValues(version.GetString()).Set(1)

			rp, err := proxy.NewMultiHostReverseProxy(
				proxy.Logger(logger),
				proxy.Config(cfg),
			)

			lh := StaticRouteHandler{
				prefix:        cfg.HTTP.Root,
				userInfoCache: userInfoCache,
				logger:        logger,
				config:        *cfg,
				oidcClient:    oidcClient,
				proxy:         rp,
			}
			if err != nil {
				return fmt.Errorf("failed to initialize reverse proxy: %w", err)
			}

			{
				middlewares := loadMiddlewares(logger, cfg, userInfoCache, signingKeyStore, traceProvider, *m)

				server, err := proxyHTTP.Server(
					proxyHTTP.Handler(lh.handler()),
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

				gr.Add(func() error {
					return server.Run()
				}, func(err error) {
					logger.Error().
						Err(err).
						Str("server", "http").
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
					logger.Error().Err(err).Str("server", "debug").Msg("Failed to initialize server")
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

// StaticRouteHandler defines a Route Handler for static routes
type StaticRouteHandler struct {
	prefix        string
	proxy         http.Handler
	userInfoCache microstore.Store
	logger        log.Logger
	config        config.Config
	oidcClient    oidc.OIDCClient
}

func (h *StaticRouteHandler) handler() http.Handler {
	m := chi.NewMux()
	m.Route(h.prefix, func(r chi.Router) {
		// Wrapper for backchannel logout
		r.Post("/backchannel_logout", h.backchannelLogout)

		// TODO: migrate oidc well knowns here in a second wrapper

		// Send all requests to the proxy handler
		r.HandleFunc("/*", h.proxy.ServeHTTP)
	})

	// Also send requests for methods unknown to chi to the proxy handler as well
	m.MethodNotAllowed(h.proxy.ServeHTTP)

	return m
}

type jse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// handle backchannel logout requests as per https://openid.net/specs/openid-connect-backchannel-1_0.html#BCRequest
func (h *StaticRouteHandler) backchannelLogout(w http.ResponseWriter, r *http.Request) {
	// parse the application/x-www-form-urlencoded POST request
	logger := h.logger.SubloggerWithRequestID(r.Context())
	if err := r.ParseForm(); err != nil {
		logger.Warn().Err(err).Msg("ParseForm failed")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: err.Error()})
		return
	}

	logoutToken, err := h.oidcClient.VerifyLogoutToken(r.Context(), r.PostFormValue("logout_token"))
	if err != nil {
		logger.Warn().Err(err).Msg("VerifyLogoutToken failed")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: err.Error()})
		return
	}

	records, err := h.userInfoCache.Read(logoutToken.SessionId)
	if errors.Is(err, microstore.ErrNotFound) || len(records) == 0 {
		render.Status(r, http.StatusOK)
		render.JSON(w, r, nil)
		return
	}

	if err != nil {
		logger.Error().Err(err).Msg("Error reading userinfo cache")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: err.Error()})
		return
	}

	for _, record := range records {
		err = h.userInfoCache.Delete(string(record.Value))
		if err != nil && !errors.Is(err, microstore.ErrNotFound) {
			// Spec requires us to return a 400 BadRequest when the session could not be destroyed
			logger.Err(err).Msg("could not delete user info from cache")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: err.Error()})
			return
		}
		logger.Debug().Msg("Deleted userinfo from cache")
	}

	// we can ignore errors when cleaning up the lookup table
	err = h.userInfoCache.Delete(logoutToken.SessionId)
	if err != nil {
		logger.Debug().Err(err).Msg("Failed to cleanup sessionid lookup entry")
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, nil)
}

func loadMiddlewares(logger log.Logger, cfg *config.Config,
	userInfoCache, signingKeyStore microstore.Store, traceProvider trace.TracerProvider, metrics metrics.Metrics) alice.Chain {
	rolesClient := settingssvc.NewRoleService("com.owncloud.api.settings", cfg.GrpcClient)
	policiesProviderClient := policiessvc.NewPoliciesProviderService("com.owncloud.api.policies", cfg.GrpcClient)

	reg := registry.GetRegistry()

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

	serviceSelector := selector.NewSelector(selector.Registry(reg))

	var userProvider backend.UserBackend
	switch cfg.AccountBackend {
	case "cs3":
		userProvider = backend.NewCS3UserBackend(
			backend.WithLogger(logger),
			backend.WithRevaGatewaySelector(gatewaySelector),
			backend.WithSelector(serviceSelector),
			backend.WithMachineAuthAPIKey(cfg.MachineAuthAPIKey),
			backend.WithOIDCissuer(cfg.OIDC.Issuer),
			backend.WithServiceAccount(cfg.ServiceAccount),
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
		middleware.OIDCWellKnownRewrite(
			logger, cfg.OIDC.Issuer,
			cfg.OIDC.RewriteWellKnown,
			oidcHTTPClient,
		),
		router.Middleware(serviceSelector, cfg.PolicySelector, cfg.Policies, logger),
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
