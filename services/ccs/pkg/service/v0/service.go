package svc

import (
	"context"
	"errors"
	"fmt"
	"github.com/DeepDiver1975/go-webdav"
	"github.com/DeepDiver1975/go-webdav/caldav"
	"github.com/DeepDiver1975/go-webdav/carddav"
	revaContext "github.com/cs3org/reva/v2/pkg/ctx"
	"github.com/cs3org/reva/v2/pkg/storage/utils/metadata"
	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/owncloud/ocis/v2/ocis-pkg/account"
	ocismiddleware "github.com/owncloud/ocis/v2/ocis-pkg/middleware"
	"github.com/owncloud/ocis/v2/services/ccs/pkg/config"
	"github.com/owncloud/ocis/v2/services/ccs/pkg/storage"
	"net/http"
	"strings"
)

type userPrincipalBackend struct{}

func (u *userPrincipalBackend) CurrentUserPrincipal(ctx context.Context) (string, error) {
	user, ok := revaContext.ContextGetUser(ctx)
	if !ok {
		return "", errors.New("no user in context")
	}

	return fmt.Sprintf("/ccs/principals/users/%s/", user.Username), nil
}

type groupwareHandler struct {
	upBackend      userPrincipalBackend
	caldavBackend  caldav.Backend
	carddavBackend carddav.Backend
}

func (u *groupwareHandler) handleOptions(w http.ResponseWriter, r *http.Request) error {
	caps := []string{"1", "3", "calendar-access", "addressbook"}
	allow := []string{"PROPFIND"}

	w.Header().Add("DAV", strings.Join(caps, ", "))
	w.Header().Add("Allow", strings.Join(allow, ", "))
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (u *groupwareHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		err := u.handleOptions(w, r)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
		return
	}

	userPrincipalPath, err := u.upBackend.CurrentUserPrincipal(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	// Home-Set Folders Discovery
	var homeSets []webdav.BackendSuppliedHomeSet
	path, err := u.caldavBackend.CalendarHomeSetPath(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	} else {
		homeSets = append(homeSets, caldav.NewCalendarHomeSet(path))
	}
	path, err = u.carddavBackend.AddressBookHomeSetPath(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	} else {
		homeSets = append(homeSets, carddav.NewAddressBookHomeSet(path))
	}

	if r.URL.Path == userPrincipalPath {
		opts := webdav.ServePrincipalOptions{
			CurrentUserPrincipalPath: userPrincipalPath,
			HomeSets:                 homeSets,
			Capabilities: []webdav.Capability{
				carddav.CapabilityAddressBook,
				caldav.CapabilityCalendar,
			},
		}

		webdav.ServePrincipal(w, r, &opts)
		return
	}
	// Current User Principal Discovery
	opts := webdav.ServePrincipalOptions{
		CurrentUserPrincipalPath: userPrincipalPath,
	}
	webdav.ServePrincipal(w, r, &opts)
	return
}

type wellknownHandler struct {
}

func (h *wellknownHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/ccs/", http.StatusMovedPermanently)
}

// NewService returns a service implementation for Service.
func NewService(opts ...Option) (*chi.Mux, error) {
	options := newOptions(opts...)

	m := chi.NewMux()
	m.Use(options.Middleware...)
	m.Use(ocismiddleware.ExtractAccountUUID(
		account.JWTSecret(options.Config.JWTSecret),
	),
	)

	conf := options.Config.Storage
	upBackend, caldavBackend, carddavBackend, err := InitStorage(options.Config.Context, conf)
	if err != nil {
		return nil, err
	}

	handler := groupwareHandler{
		upBackend:      *upBackend,
		caldavBackend:  caldavBackend,
		carddavBackend: carddavBackend,
	}
	caldavHandler := caldav.Handler{Backend: caldavBackend, Prefix: "/ccs"}
	carddavHandler := carddav.Handler{Backend: carddavBackend, Prefix: "/ccs"}
	wellknownHandler := wellknownHandler{}

	m.Route(options.Config.HTTP.Root, func(r chi.Router) {
		r.Use(middleware.StripSlashes)
		r.Mount("/.well-known/caldav", &wellknownHandler)
		r.Mount("/.well-known/carddav", &wellknownHandler)
		r.Mount("/ccs", &handler)
		r.Mount("/ccs/principals", &handler)
		r.Mount("/ccs/calendars", &caldavHandler)
		r.Mount("/ccs/addressbooks", &carddavHandler)
	})

	_ = chi.Walk(m, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		options.Logger.Debug().Str("method", method).Str("route", route).Int("middlewares", len(middlewares)).Msg("serving endpoint")
		return nil
	})

	return m, nil
}

func InitStorage(ctx context.Context, conf config.Storage) (*userPrincipalBackend, caldav.Backend, carddav.Backend, error) {
	s, err := metadata.NewCS3Storage(conf.GatewayAddress, conf.GatewayAddress, conf.SystemUserID, conf.SystemUserIDP, conf.SystemUserAPIKey)
	if err != nil {
		return nil, nil, nil, err
	}
	err = s.Init(ctx, "calendar-contacts-service")
	if err != nil {
		return nil, nil, nil, err
	}

	upBackend := &userPrincipalBackend{}

	caldavBackend, carddavBackend, err := storage.NewFilesystem(s, "/calendar/", "/addressbooks/", upBackend)
	return upBackend, caldavBackend, carddavBackend, err
}
