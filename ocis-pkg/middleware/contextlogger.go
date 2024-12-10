package middleware

import (
	"net/http"

	"github.com/owncloud/ocis/v2/ocis-pkg/log"
)

func NewContextLogger(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			newLogger := logger.With().
				Str(log.RequestIDString, r.Header.Get("X-Request-ID")).
				Str("proto", r.Proto).
				Str("method", r.Method).
				Str("host", r.Host).
				Str("path", r.URL.Path).
				Str("query", r.URL.RawQuery).
				Str("fragment", r.URL.Fragment).
				Str("remote-addr", r.RemoteAddr).
				Str("user-agent", r.Header.Get("User-Agent")).
				Str("content-length", r.Header.Get("Content-Length")).
				Str("content-type", r.Header.Get("Content-Type")).
				Logger()

			next.ServeHTTP(w, r.WithContext(newLogger.WithContext(r.Context())))
		})
	}
}
