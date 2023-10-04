package svc

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/cs3org/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/cs3org/reva/v2/pkg/storage/utils/templates"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/riandyrn/otelchi"
	merrors "go-micro.dev/v4/errors"
	grpcmetadata "google.golang.org/grpc/metadata"

	"github.com/owncloud/ocis/v2/ocis-pkg/log"
	"github.com/owncloud/ocis/v2/ocis-pkg/registry"
	"github.com/owncloud/ocis/v2/ocis-pkg/tracing"
	thumbnailsmsg "github.com/owncloud/ocis/v2/protogen/gen/ocis/messages/thumbnails/v0"
	searchsvc "github.com/owncloud/ocis/v2/protogen/gen/ocis/services/search/v0"
	thumbnailssvc "github.com/owncloud/ocis/v2/protogen/gen/ocis/services/thumbnails/v0"
	"github.com/owncloud/ocis/v2/services/webdav/pkg/config"
	"github.com/owncloud/ocis/v2/services/webdav/pkg/constants"
	"github.com/owncloud/ocis/v2/services/webdav/pkg/dav/requests"
)

func init() {
	// register method with chi before any routing is set up
	chi.RegisterMethod("REPORT")
}

const (
	TokenHeader = "X-Access-Token"
)

var (
	codesEnum = map[int]string{
		http.StatusBadRequest:       "Sabre\\DAV\\Exception\\BadRequest",
		http.StatusUnauthorized:     "Sabre\\DAV\\Exception\\NotAuthenticated",
		http.StatusNotFound:         "Sabre\\DAV\\Exception\\NotFound",
		http.StatusMethodNotAllowed: "Sabre\\DAV\\Exception\\MethodNotAllowed",
	}
)

// Service defines the extension handlers.
type Service interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
	Thumbnail(http.ResponseWriter, *http.Request)
}

// NewService returns a service implementation for Service.
func NewService(opts ...Option) (Service, error) {
	options := newOptions(opts...)
	conf := options.Config

	m := chi.NewMux()
	m.Use(
		otelchi.Middleware(
			conf.Service.Name,
			otelchi.WithChiRoutes(m),
			otelchi.WithTracerProvider(options.TraceProvider),
			otelchi.WithPropagators(tracing.GetPropagator()),
		),
	)

	tm, err := pool.StringToTLSMode(conf.GRPCClientTLS.Mode)
	if err != nil {
		return nil, err
	}
	gatewaySelector, err := pool.GatewaySelector(conf.RevaGateway,
		pool.WithTLSCACert(conf.GRPCClientTLS.CACert),
		pool.WithTLSMode(tm),
		pool.WithRegistry(registry.GetRegistry()),
		pool.WithTracerProvider(options.TraceProvider),
	)
	if err != nil {
		return nil, err
	}

	svc := Webdav{
		config:           conf,
		log:              options.Logger,
		mux:              m,
		searchClient:     searchsvc.NewSearchProviderService("com.owncloud.api.search", conf.GrpcClient),
		thumbnailsClient: thumbnailssvc.NewThumbnailService("com.owncloud.api.thumbnails", conf.GrpcClient),
		gatewaySelector:  gatewaySelector,
	}

	if svc.config.DisablePreviews {
		svc.thumbnailsClient = nil
	}
	m.Route(options.Config.HTTP.Root, func(r chi.Router) {

		if !svc.config.DisablePreviews {
			r.Group(func(r chi.Router) {
				r.Use(svc.DavUserContext())

				r.Get("/remote.php/dav/spaces/{id}", svc.SpacesThumbnail)
				r.Get("/remote.php/dav/spaces/{id}/*", svc.SpacesThumbnail)
				r.Get("/dav/spaces/{id}", svc.SpacesThumbnail)
				r.Get("/dav/spaces/{id}/*", svc.SpacesThumbnail)
				r.MethodFunc("REPORT", "/remote.php/dav/spaces*", svc.Search)
				r.MethodFunc("REPORT", "/dav/spaces*", svc.Search)

				r.Get("/remote.php/dav/files/{id}", svc.Thumbnail)
				r.Get("/remote.php/dav/files/{id}/*", svc.Thumbnail)
				r.Get("/dav/files/{id}", svc.Thumbnail)
				r.Get("/dav/files/{id}/*", svc.Thumbnail)

				r.MethodFunc("REPORT", "/remote.php/dav/files*", svc.Search)
				r.MethodFunc("REPORT", "/dav/files*", svc.Search)
			})

			r.Group(func(r chi.Router) {
				r.Use(svc.DavPublicContext())

				r.Head("/remote.php/dav/public-files/{token}/*", svc.PublicThumbnailHead)
				r.Head("/dav/public-files/{token}/*", svc.PublicThumbnailHead)

				r.Get("/remote.php/dav/public-files/{token}/*", svc.PublicThumbnail)
				r.Get("/dav/public-files/{token}/*", svc.PublicThumbnail)
			})

			r.Group(func(r chi.Router) {
				r.Use(svc.WebDAVContext())
				r.Get("/remote.php/webdav/*", svc.Thumbnail)
				r.Get("/webdav/*", svc.Thumbnail)

				r.MethodFunc("REPORT", "/remote.php/webdav*", svc.Search)
				r.MethodFunc("REPORT", "/webdav*", svc.Search)
			})
		}

	})

	_ = chi.Walk(m, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		options.Logger.Debug().Str("method", method).Str("route", route).Int("middlewares", len(middlewares)).Msg("serving endpoint")
		return nil
	})

	return svc, nil
}

// Webdav implements the business logic for Service.
type Webdav struct {
	config           *config.Config
	log              log.Logger
	mux              *chi.Mux
	searchClient     searchsvc.SearchProviderService
	thumbnailsClient thumbnailssvc.ThumbnailService
	gatewaySelector  pool.Selectable[gatewayv1beta1.GatewayAPIClient]
}

// ServeHTTP implements the Service interface.
func (g Webdav) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mux.ServeHTTP(w, r)
}

func (g Webdav) DavUserContext() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			filePath := r.URL.Path

			id := chi.URLParam(r, "id")
			id, err := url.QueryUnescape(id)
			if err == nil && id != "" {
				ctx = context.WithValue(ctx, constants.ContextKeyID, id)
			}

			if id != "" {
				filePath = strings.TrimPrefix(filePath, path.Join("/remote.php/dav/spaces", id))
				filePath = strings.TrimPrefix(filePath, path.Join("/dav/spaces", id))

				filePath = strings.TrimPrefix(filePath, path.Join("/remote.php/dav/files", id))
				filePath = strings.TrimPrefix(filePath, path.Join("/dav/files", id))
				filePath = strings.TrimPrefix(filePath, "/")
			}

			ctx = context.WithValue(ctx, constants.ContextKeyPath, filePath)

			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}

func (g Webdav) DavPublicContext() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			filePath := r.URL.Path

			if token := chi.URLParam(r, "token"); token != "" {
				filePath = strings.TrimPrefix(filePath, path.Join("/remote.php/dav/public-files", token)+"/")
				filePath = strings.TrimPrefix(filePath, path.Join("/dav/public-files", token)+"/")
			}
			ctx = context.WithValue(ctx, constants.ContextKeyPath, filePath)

			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}
func (g Webdav) WebDAVContext() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			filePath := r.URL.Path
			filePath = strings.TrimPrefix(filePath, "/remote.php")
			filePath = strings.TrimPrefix(filePath, "/webdav/")

			ctx := context.WithValue(r.Context(), constants.ContextKeyPath, filePath)

			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}

// SpacesThumbnail is the endpoint for retrieving thumbnails inside of spaces.
func (g Webdav) SpacesThumbnail(w http.ResponseWriter, r *http.Request) {
	logger := g.log.SubloggerWithRequestID(r.Context())
	tr, err := requests.ParseThumbnailRequest(r)
	if err != nil {
		logger.Debug().Err(err).Msg("could not create Request")
		renderError(w, r, errBadRequest(err.Error()))
		return
	}
	t := r.Header.Get(TokenHeader)

	fullPath := filepath.Join(tr.Identifier, tr.Filepath)
	rsp, err := g.thumbnailsClient.GetThumbnail(r.Context(), &thumbnailssvc.GetThumbnailRequest{
		Filepath:      strings.TrimLeft(tr.Filepath, "/"),
		ThumbnailType: extensionToThumbnailType(strings.TrimLeft(tr.Extension, ".")),
		Width:         tr.Width,
		Height:        tr.Height,
		Processor:     tr.Processor,
		Source: &thumbnailssvc.GetThumbnailRequest_Cs3Source{
			Cs3Source: &thumbnailsmsg.CS3Source{
				Path:          fullPath,
				Authorization: t,
			},
		},
	})
	if err != nil {
		e := merrors.Parse(err.Error())
		switch e.Code {
		case http.StatusNotFound:
			// StatusNotFound is expected for unsupported files
			renderError(w, r, errNotFound(notFoundMsg(tr.Filename)))
			return
		case http.StatusTooEarly:
			// StatusTooEarly if file is processing
			renderError(w, r, errTooEarly(err.Error()))
			return
		case http.StatusBadRequest:
			renderError(w, r, errBadRequest(e.Detail))
		default:
			renderError(w, r, errInternalError(err.Error()))
		}
		logger.Debug().Err(err).Msg("could not get thumbnail")
		return
	}

	g.sendThumbnailResponse(rsp, w, r)
}

// Thumbnail implements the Service interface.
func (g Webdav) Thumbnail(w http.ResponseWriter, r *http.Request) {
	logger := g.log.SubloggerWithRequestID(r.Context())
	tr, err := requests.ParseThumbnailRequest(r)
	if err != nil {
		logger.Debug().Err(err).Msg("could not create Request")
		renderError(w, r, errBadRequest(err.Error()))
		return
	}

	t := r.Header.Get(TokenHeader)

	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		logger.Error().Err(err).Msg("could not get reva gatewayClient")
		renderError(w, r, errInternalError("could not get reva gatewayClient"))
		return
	}

	var user *userv1beta1.User
	if tr.Identifier == "" {
		// look up user from token via WhoAmI
		userRes, err := gatewayClient.WhoAmI(r.Context(), &gatewayv1beta1.WhoAmIRequest{
			Token: t,
		})
		if err != nil {
			logger.Error().Err(err).Msg("could not get user: transport error")
			renderError(w, r, errInternalError("could not get user"))
			return
		}
		if userRes.Status.Code != rpcv1beta1.Code_CODE_OK {
			logger.Debug().Str("grpcmessage", userRes.GetStatus().GetMessage()).Msg("could not get user")
			renderError(w, r, errInternalError("could not get user"))
			return
		}
		user = userRes.GetUser()
	} else {
		// look up user from URL via GetUserByClaim
		ctx := grpcmetadata.AppendToOutgoingContext(r.Context(), TokenHeader, t)
		userRes, err := gatewayClient.GetUserByClaim(ctx, &userv1beta1.GetUserByClaimRequest{
			Claim: "username",
			Value: tr.Identifier,
		})
		if err != nil {
			logger.Error().Err(err).Msg("could not get user: transport error")
			renderError(w, r, errInternalError("could not get user"))
			return
		}
		if userRes.Status.Code != rpcv1beta1.Code_CODE_OK {
			logger.Debug().Str("grpcmessage", userRes.GetStatus().GetMessage()).Msg("could not get user")
			renderError(w, r, errInternalError("could not get user"))
			return
		}
		user = userRes.GetUser()
	}

	fullPath := filepath.Join(templates.WithUser(user, g.config.WebdavNamespace), tr.Filepath)
	rsp, err := g.thumbnailsClient.GetThumbnail(r.Context(), &thumbnailssvc.GetThumbnailRequest{
		Filepath:      strings.TrimLeft(tr.Filepath, "/"),
		ThumbnailType: extensionToThumbnailType(strings.TrimLeft(tr.Extension, ".")),
		Width:         tr.Width,
		Height:        tr.Height,
		Processor:     tr.Processor,
		Source: &thumbnailssvc.GetThumbnailRequest_Cs3Source{
			Cs3Source: &thumbnailsmsg.CS3Source{
				Path:          fullPath,
				Authorization: t,
			},
		},
	})
	if err != nil {
		e := merrors.Parse(err.Error())
		switch e.Code {
		case http.StatusNotFound:
			// StatusNotFound is expected for unsupported files
			renderError(w, r, errNotFound(notFoundMsg(tr.Filename)))
			return
		case http.StatusBadRequest:
			renderError(w, r, errBadRequest(e.Detail))
		default:
			renderError(w, r, errInternalError(err.Error()))
		}
		g.log.Error().Err(err).Msg("could not get thumbnail")
		return
	}

	g.sendThumbnailResponse(rsp, w, r)
}

func (g Webdav) PublicThumbnail(w http.ResponseWriter, r *http.Request) {
	logger := g.log.SubloggerWithRequestID(r.Context())
	tr, err := requests.ParseThumbnailRequest(r)
	if err != nil {
		logger.Debug().Err(err).Msg("could not create Request")
		renderError(w, r, errBadRequest(err.Error()))
		return
	}

	rsp, err := g.thumbnailsClient.GetThumbnail(r.Context(), &thumbnailssvc.GetThumbnailRequest{
		Filepath:      strings.TrimLeft(tr.Filepath, "/"),
		ThumbnailType: extensionToThumbnailType(strings.TrimLeft(tr.Extension, ".")),
		Width:         tr.Width,
		Height:        tr.Height,
		Processor:     tr.Processor,
		Source: &thumbnailssvc.GetThumbnailRequest_WebdavSource{
			WebdavSource: &thumbnailsmsg.WebdavSource{
				Url:             g.config.OcisPublicURL + r.URL.RequestURI(),
				IsPublicLink:    true,
				PublicLinkToken: tr.PublicLinkToken,
			},
		},
	})
	if err != nil {
		e := merrors.Parse(err.Error())
		switch e.Code {
		case http.StatusNotFound:
			// StatusNotFound is expected for unsupported files
			renderError(w, r, errNotFound(notFoundMsg(tr.Filename)))
			return
		case http.StatusBadRequest:
			renderError(w, r, errBadRequest(e.Detail))
		default:
			renderError(w, r, errInternalError(err.Error()))
		}
		g.log.Error().Err(err).Msg("could not get thumbnail")
		return
	}

	g.sendThumbnailResponse(rsp, w, r)
}

func (g Webdav) PublicThumbnailHead(w http.ResponseWriter, r *http.Request) {
	logger := g.log.SubloggerWithRequestID(r.Context())
	tr, err := requests.ParseThumbnailRequest(r)
	if err != nil {
		logger.Debug().Err(err).Msg("could not create Request")
		renderError(w, r, errBadRequest(err.Error()))
		return
	}

	_, err = g.thumbnailsClient.GetThumbnail(r.Context(), &thumbnailssvc.GetThumbnailRequest{
		Filepath:      strings.TrimLeft(tr.Filepath, "/"),
		ThumbnailType: extensionToThumbnailType(strings.TrimLeft(tr.Extension, ".")),
		Width:         tr.Width,
		Height:        tr.Height,
		Processor:     tr.Processor,
		Source: &thumbnailssvc.GetThumbnailRequest_WebdavSource{
			WebdavSource: &thumbnailsmsg.WebdavSource{
				Url:             g.config.OcisPublicURL + r.URL.RequestURI(),
				IsPublicLink:    true,
				PublicLinkToken: tr.PublicLinkToken,
			},
		},
	})
	if err != nil {
		e := merrors.Parse(err.Error())
		switch e.Code {
		case http.StatusNotFound:
			// StatusNotFound is expected for unsupported files
			renderError(w, r, errNotFound(notFoundMsg(tr.Filename)))
			return
		case http.StatusBadRequest:
			renderError(w, r, errBadRequest(e.Detail))
		default:
			renderError(w, r, errInternalError(err.Error()))
		}
		logger.Debug().Err(err).Msg("could not get thumbnail")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (g Webdav) sendThumbnailResponse(rsp *thumbnailssvc.GetThumbnailResponse, w http.ResponseWriter, r *http.Request) {
	logger := g.log.SubloggerWithRequestID(r.Context())
	client := &http.Client{
		// Timeout: time.Second * 5,
	}

	dlReq, err := http.NewRequest(http.MethodGet, rsp.DataEndpoint, http.NoBody)
	if err != nil {
		renderError(w, r, errInternalError(err.Error()))
		logger.Error().Err(err).Msg("could not create download thumbnail request")
		return
	}
	dlReq.Header.Set("Transfer-Token", rsp.TransferToken)

	dlRsp, err := client.Do(dlReq)
	if err != nil {
		renderError(w, r, errInternalError(err.Error()))
		logger.Error().Err(err).Msg("could not download thumbnail: transport error")
		return
	}
	defer dlRsp.Body.Close()

	if dlRsp.StatusCode != http.StatusOK {
		logger.Debug().
			Str("transfer_token", rsp.TransferToken).
			Str("data_endpoint", rsp.DataEndpoint).
			Str("response_status", dlRsp.Status).
			Msg("could not download thumbnail")
		renderError(w, r, errInternalError("could not download thumbnail"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", dlRsp.Header.Get("Content-Type"))
	_, err = io.Copy(w, dlRsp.Body)
	if err != nil {
		logger.Error().Err(err).Msg("failed to write thumbnail to response writer")
	}
}

func extensionToThumbnailType(ext string) thumbnailsmsg.ThumbnailType {
	switch strings.ToUpper(ext) {
	case "GIF":
		return thumbnailsmsg.ThumbnailType_GIF
	case "PNG":
		return thumbnailsmsg.ThumbnailType_PNG
	default:
		return thumbnailsmsg.ThumbnailType_JPG
	}
}

// http://www.webdav.org/specs/rfc4918.html#ELEMENT_error
type errResponse struct {
	HTTPStatusCode int      `json:"-" xml:"-"`
	XMLName        xml.Name `xml:"d:error"`
	Xmlnsd         string   `xml:"xmlns:d,attr"`
	Xmlnss         string   `xml:"xmlns:s,attr"`
	Exception      string   `xml:"s:exception"`
	Message        string   `xml:"s:message"`
	InnerXML       []byte   `xml:",innerxml"`
}

func newErrResponse(statusCode int, msg string) *errResponse {
	rsp := &errResponse{
		HTTPStatusCode: statusCode,
		Xmlnsd:         "DAV",
		Xmlnss:         "http://sabredav.org/ns",
		Exception:      codesEnum[statusCode],
	}
	if msg != "" {
		rsp.Message = msg
	}
	return rsp
}

func errInternalError(msg string) *errResponse {
	return newErrResponse(http.StatusInternalServerError, msg)
}

func errBadRequest(msg string) *errResponse {
	return newErrResponse(http.StatusBadRequest, msg)
}

func errNotFound(msg string) *errResponse {
	return newErrResponse(http.StatusNotFound, msg)
}

func errTooEarly(msg string) *errResponse {
	return newErrResponse(http.StatusTooEarly, msg)
}

func renderError(w http.ResponseWriter, r *http.Request, err *errResponse) {
	render.Status(r, err.HTTPStatusCode)
	render.XML(w, r, err)
}

func notFoundMsg(name string) string {
	return "File with name " + name + " could not be located"
}
