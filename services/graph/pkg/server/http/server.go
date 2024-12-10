package http

import (
	"fmt"
	stdhttp "net/http"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/cs3org/reva/v2/pkg/events/stream"
	"github.com/cs3org/reva/v2/pkg/rgrpc/todo/pool"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/pkg/errors"
	"go-micro.dev/v4"
	"go-micro.dev/v4/events"

	"github.com/owncloud/ocis/v2/ocis-pkg/account"
	"github.com/owncloud/ocis/v2/ocis-pkg/cors"
	"github.com/owncloud/ocis/v2/ocis-pkg/keycloak"
	"github.com/owncloud/ocis/v2/ocis-pkg/middleware"
	"github.com/owncloud/ocis/v2/ocis-pkg/registry"
	"github.com/owncloud/ocis/v2/ocis-pkg/service/grpc"
	"github.com/owncloud/ocis/v2/ocis-pkg/service/http"
	"github.com/owncloud/ocis/v2/ocis-pkg/version"
	ehsvc "github.com/owncloud/ocis/v2/protogen/gen/ocis/services/eventhistory/v0"
	searchsvc "github.com/owncloud/ocis/v2/protogen/gen/ocis/services/search/v0"
	settingssvc "github.com/owncloud/ocis/v2/protogen/gen/ocis/services/settings/v0"
	graphMiddleware "github.com/owncloud/ocis/v2/services/graph/pkg/middleware"
	svc "github.com/owncloud/ocis/v2/services/graph/pkg/service/v0"
)

// Server initializes the http service and server.
func Server(opts ...Option) (http.Service, error) {
	options := newOptions(opts...)

	service, err := http.NewService(
		http.TLSConfig(options.Config.HTTP.TLS),
		http.Logger(options.Logger),
		http.Namespace(options.Config.HTTP.Namespace),
		http.Name(options.Config.Service.Name),
		http.Version(version.GetString()),
		http.Address(options.Config.HTTP.Addr),
		http.Context(options.Context),
		http.Flags(options.Flags...),
		http.TraceProvider(options.TraceProvider),
	)
	if err != nil {
		options.Logger.Error().
			Err(err).
			Msg("Error initializing http service")
		return http.Service{}, fmt.Errorf("could not initialize http service: %w", err)
	}

	var eventsStream events.Stream

	if options.Config.Events.Endpoint != "" {
		var err error
		eventsStream, err = stream.NatsFromConfig(options.Config.Service.Name, false, stream.NatsConfig(options.Config.Events))
		if err != nil {
			options.Logger.Error().
				Err(err).
				Msg("Error initializing events publisher")
			return http.Service{}, errors.Wrap(err, "could not initialize events publisher")
		}
	}

	middlewares := []func(stdhttp.Handler) stdhttp.Handler{
		middleware.TraceContext,
		chimiddleware.RequestID,
		middleware.Version(
			options.Config.Service.Name,
			version.GetString(),
		),
		middleware.Logger(
			options.Logger,
		),
		middleware.Cors(
			cors.Logger(options.Logger),
			cors.AllowedOrigins(options.Config.HTTP.CORS.AllowedOrigins),
			cors.AllowedMethods(options.Config.HTTP.CORS.AllowedMethods),
			cors.AllowedHeaders(options.Config.HTTP.CORS.AllowedHeaders),
			cors.AllowCredentials(options.Config.HTTP.CORS.AllowCredentials),
		),
	}
	// how do we secure the api?
	var requireAdminMiddleware func(stdhttp.Handler) stdhttp.Handler
	var roleService svc.RoleService
	var valueService settingssvc.ValueService
	var gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
	grpcClient, err := grpc.NewClient(
		append(
			grpc.GetClientOptions(options.Config.GRPCClientTLS),
			grpc.WithTraceProvider(options.TraceProvider),
			grpc.WithClientNameAndVersion(options.Config.Service.Name, version.GetString()),
		)...)
	if err != nil {
		return http.Service{}, err
	}
	if options.Config.HTTP.APIToken == "" {
		middlewares = append(middlewares,
			graphMiddleware.Auth(
				account.Logger(options.Logger),
				account.JWTSecret(options.Config.TokenManager.JWTSecret),
			))
		roleService = settingssvc.NewRoleService("com.owncloud.api.settings", grpcClient)
		valueService = settingssvc.NewValueService("com.owncloud.api.settings", grpcClient)
		gatewaySelector, err = pool.GatewaySelector(
			options.Config.Reva.Address,
			append(
				options.Config.Reva.GetRevaOptions(),
				pool.WithRegistry(registry.GetRegistry()),
				pool.WithTracerProvider(options.TraceProvider),
			)...)
		if err != nil {
			return http.Service{}, errors.Wrap(err, "could not initialize gateway selector")
		}
	} else {
		middlewares = append(middlewares, graphMiddleware.Token(options.Config.HTTP.APIToken))
		// use a dummy admin middleware for the chi router
		requireAdminMiddleware = func(next stdhttp.Handler) stdhttp.Handler {
			return next
		}
		// no gateway client needed
	}

	// Keycloak client is optional, so if it stays nil, it's fine.
	var keyCloakClient keycloak.Client
	if options.Config.Keycloak.BasePath != "" {
		kcc := options.Config.Keycloak
		if kcc.ClientID == "" || kcc.ClientSecret == "" || kcc.ClientRealm == "" || kcc.UserRealm == "" {
			return http.Service{}, errors.New("keycloak client id, secret, client realm and user realm must be set")
		}
		keyCloakClient = keycloak.New(kcc.BasePath, kcc.ClientID, kcc.ClientSecret, kcc.ClientRealm, kcc.InsecureSkipVerify)
	}

	hClient := ehsvc.NewEventHistoryService("com.owncloud.api.eventhistory", grpcClient)

	var handle svc.Service
	handle, err = svc.NewService(
		svc.Context(options.Context),
		svc.Logger(options.Logger),
		svc.Config(options.Config),
		svc.Middleware(middlewares...),
		svc.EventsPublisher(eventsStream),
		svc.EventsConsumer(eventsStream),
		svc.WithRoleService(roleService),
		svc.WithValueService(valueService),
		svc.WithRequireAdminMiddleware(requireAdminMiddleware),
		svc.WithGatewaySelector(gatewaySelector),
		svc.WithSearchService(searchsvc.NewSearchProviderService("com.owncloud.api.search", grpcClient)),
		svc.KeycloakClient(keyCloakClient),
		svc.EventHistoryClient(hClient),
		svc.TraceProvider(options.TraceProvider),
	)

	if err != nil {
		return http.Service{}, errors.New("could not initialize graph service")
	}

	if err := micro.RegisterHandler(service.Server(), handle); err != nil {
		return http.Service{}, err
	}

	return service, nil
}
