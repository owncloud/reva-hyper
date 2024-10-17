package command

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/oklog/run"
	"github.com/owncloud/ocis/v2/ocis-pkg/config/configlog"
	"github.com/owncloud/ocis/v2/ocis-pkg/tracing"
	"github.com/owncloud/ocis/v2/ocis-pkg/version"
	"github.com/owncloud/ocis/v2/services/idp/pkg/config"
	"github.com/owncloud/ocis/v2/services/idp/pkg/config/parser"
	"github.com/owncloud/ocis/v2/services/idp/pkg/logging"
	"github.com/owncloud/ocis/v2/services/idp/pkg/metrics"
	"github.com/owncloud/ocis/v2/services/idp/pkg/server/debug"
	"github.com/owncloud/ocis/v2/services/idp/pkg/server/http"
	"github.com/urfave/cli/v2"
)

const _rsaKeySize = 4096

// Server is the entrypoint for the server command.
func Server(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "server",
		Usage:    fmt.Sprintf("start the %s service without runtime (unsupervised mode)", cfg.Service.Name),
		Category: "server",
		Before: func(c *cli.Context) error {
			configlog.ReturnFatal(parser.ParseConfig(cfg))

			if cfg.IDP.EncryptionSecretFile != "" {
				if err := ensureEncryptionSecretExists(cfg.IDP.EncryptionSecretFile); err != nil {
					return err
				}
				if err := ensureSigningPrivateKeyExists(cfg.IDP.SigningPrivateKeyFiles); err != nil {
					return err
				}
			}
			return nil
		},
		Action: func(c *cli.Context) error {
			logger := logging.Configure(cfg.Service.Name, cfg.Log)
			traceProvider, err := tracing.GetServiceTraceProvider(cfg.Tracing, cfg.Service.Name)
			if err != nil {
				return err
			}
			var (
				gr          = run.Group{}
				ctx, cancel = context.WithCancel(c.Context)
				metrics     = metrics.New()
			)

			defer cancel()

			metrics.BuildInfo.WithLabelValues(version.GetString()).Set(1)

			{
				server, err := http.Server(
					http.Logger(logger),
					http.Context(ctx),
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
					return server.Run()
				}, func(err error) {
					logger.Error().
						Str("transport", "http").
						Err(err).
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

func ensureEncryptionSecretExists(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		// If the file exists we can just return
		return nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0o700)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	secret := make([]byte, 32)
	_, err = rand.Read(secret)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, bytes.NewReader(secret))
	if err != nil {
		return err
	}

	return nil
}

func ensureSigningPrivateKeyExists(paths []string) error {
	for _, path := range paths {
		file, err := os.Stat(path)
		if err == nil && file.Size() > 0 {
			// If the file exists and is not empty we can just return
			return nil
		}
		if !errors.Is(err, fs.ErrNotExist) && file.Size() > 0 {
			return err
		}

		dir := filepath.Dir(path)
		err = os.MkdirAll(dir, 0o700)
		if err != nil {
			return err
		}

		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		defer f.Close()

		pk, err := rsa.GenerateKey(rand.Reader, _rsaKeySize)
		if err != nil {
			return err
		}

		pb := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(pk),
		}
		if err := pem.Encode(f, pb); err != nil {
			return err
		}
	}
	return nil
}
