package service

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/cs3org/reva/v2/pkg/bytesize"
	ctxpkg "github.com/cs3org/reva/v2/pkg/ctx"
	"github.com/cs3org/reva/v2/pkg/events"
	"github.com/cs3org/reva/v2/pkg/events/stream"
	"github.com/cs3org/reva/v2/pkg/rhttp"
	"go.opentelemetry.io/otel/trace"

	"github.com/owncloud/ocis/v2/ocis-pkg/log"
	"github.com/owncloud/ocis/v2/services/antivirus/pkg/config"
	"github.com/owncloud/ocis/v2/services/antivirus/pkg/scanners"
)

var (
	// ErrFatal is returned when a fatal error occurs, and we want to exit.
	ErrFatal = errors.New("fatal error")
	// ErrEvent is returned when something went wrong with a specific event.
	ErrEvent = errors.New("event error")
)

// Scanner is an abstraction for the actual virus scan
type Scanner interface {
	Scan(body scanners.Input) (scanners.Result, error)
}

// NewAntivirus returns a service implementation for Service.
func NewAntivirus(c *config.Config, l log.Logger, tp trace.TracerProvider) (Antivirus, error) {

	var scanner Scanner
	var err error
	switch c.Scanner.Type {
	default:
		return Antivirus{}, fmt.Errorf("unknown av scanner: '%s'", c.Scanner.Type)
	case "clamav":
		scanner = scanners.NewClamAV(c.Scanner.ClamAV.Socket)
	case "icap":
		scanner, err = scanners.NewICAP(c.Scanner.ICAP.URL, c.Scanner.ICAP.Service, c.Scanner.ICAP.Timeout)
	}
	if err != nil {
		return Antivirus{}, err
	}

	av := Antivirus{
		c:       c,
		l:       l,
		tp:      tp,
		s:       scanner,
		client:  rhttp.GetHTTPClient(rhttp.Insecure(true)),
		stopCh:  make(chan struct{}, 1),
		stopped: new(atomic.Bool),
	}

	switch o := events.PostprocessingOutcome(c.InfectedFileHandling); o {
	case events.PPOutcomeContinue, events.PPOutcomeAbort, events.PPOutcomeDelete:
		av.o = o
	default:
		return av, fmt.Errorf("unknown infected file handling '%s'", o)
	}

	if c.MaxScanSize != "" {
		b, err := bytesize.Parse(c.MaxScanSize)
		if err != nil {
			return av, err
		}

		av.m = b.Bytes()
	}

	return av, nil
}

// Antivirus defines implements the business logic for Service.
type Antivirus struct {
	c  *config.Config
	l  log.Logger
	s  Scanner
	o  events.PostprocessingOutcome
	m  uint64
	tp trace.TracerProvider

	client  *http.Client
	stopCh  chan struct{}
	stopped *atomic.Bool
}

// Run runs the service
func (av Antivirus) Run() error {
	evtsCfg := av.c.Events

	var rootCAPool *x509.CertPool
	if evtsCfg.TLSRootCACertificate != "" {
		rootCrtFile, err := os.Open(evtsCfg.TLSRootCACertificate)
		if err != nil {
			return err
		}

		var certBytes bytes.Buffer
		if _, err := io.Copy(&certBytes, rootCrtFile); err != nil {
			return err
		}

		rootCAPool = x509.NewCertPool()
		rootCAPool.AppendCertsFromPEM(certBytes.Bytes())
		evtsCfg.TLSInsecure = false
	}

	natsStream, err := stream.NatsFromConfig(av.c.Service.Name, false, stream.NatsConfig(av.c.Events))
	if err != nil {
		return err
	}

	ch, err := events.Consume(natsStream, "antivirus", events.StartPostprocessingStep{})
	if err != nil {
		return err
	}

EventLoop:
	for {
		select {
		case e, ok := <-ch:
			if !ok {
				break EventLoop
			}

			err := av.processEvent(e, natsStream)
			if err != nil {
				switch {
				case errors.Is(err, ErrFatal):
					return err
				case errors.Is(err, ErrEvent):
					// Right now logging of these happens in the processEvent method, might be cleaner to do it here.
					continue
				default:
					av.l.Fatal().Err(err).Msg("unknown error - exiting")
				}
			}

			if av.stopped.Load() {
				break EventLoop
			}
		case <-av.stopCh:
			break EventLoop
		}
	}

	return nil
}

func (av Antivirus) Close() {
	if av.stopped.CompareAndSwap(false, true) {
		close(av.stopCh)
	}
}

func (av Antivirus) processEvent(e events.Event, s events.Publisher) error {
	ctx := e.GetTraceContext(context.Background())
	ctx, span := av.tp.Tracer("antivirus").Start(ctx, "processEvent")
	defer span.End()
	av.l.Info().Str("traceID", span.SpanContext().TraceID().String()).Msg("TraceID")
	ev := e.Event.(events.StartPostprocessingStep)
	if ev.StepToStart != events.PPStepAntivirus {
		return nil
	}

	if av.c.DebugScanOutcome != "" {
		av.l.Warn().Str("antivir, clamav", ">>>>>>> ANTIVIRUS_DEBUG_SCAN_OUTCOME IS SET NO ACTUAL VIRUS SCAN IS PERFORMED!").Send()
		if err := events.Publish(ctx, s, events.PostprocessingStepFinished{
			FinishedStep:  events.PPStepAntivirus,
			Outcome:       events.PostprocessingOutcome(av.c.DebugScanOutcome),
			UploadID:      ev.UploadID,
			ExecutingUser: ev.ExecutingUser,
			Filename:      ev.Filename,
			Result: events.VirusscanResult{
				Infected:    true,
				Description: "DEBUG: forced outcome",
				Scandate:    time.Now(),
				ResourceID:  ev.ResourceID,
			},
		}); err != nil {
			av.l.Fatal().Err(err).Str("uploadid", ev.UploadID).Interface("resourceID", ev.ResourceID).Msg("cannot publish events - exiting")
			return fmt.Errorf("%w: cannot publish events", ErrFatal)
		}
		return fmt.Errorf("%w: no actual virus scan performed", ErrEvent)
	}

	av.l.Debug().Str("uploadid", ev.UploadID).Str("filename", ev.Filename).Msg("Starting virus scan.")
	var errmsg string
	res, err := av.process(ev)
	if err != nil {
		errmsg = err.Error()
	}

	var outcome events.PostprocessingOutcome
	switch {
	case res.Infected:
		outcome = av.o
	case !res.Infected && err == nil:
		outcome = events.PPOutcomeContinue
	case err != nil:
		outcome = events.PPOutcomeRetry
	default:
		// Not sure what this is about. abort.
		outcome = events.PPOutcomeAbort
	}

	av.l.Info().Str("uploadid", ev.UploadID).Interface("resourceID", ev.ResourceID).Str("virus", res.Description).Str("outcome", string(outcome)).Str("filename", ev.Filename).Str("user", ev.ExecutingUser.GetId().GetOpaqueId()).Bool("infected", res.Infected).Msg("File scanned")
	if err := events.Publish(ctx, s, events.PostprocessingStepFinished{
		FinishedStep:  events.PPStepAntivirus,
		Outcome:       outcome,
		UploadID:      ev.UploadID,
		ExecutingUser: ev.ExecutingUser,
		Filename:      ev.Filename,
		Result: events.VirusscanResult{
			Infected:    res.Infected,
			Description: res.Description,
			Scandate:    time.Now(),
			ResourceID:  ev.ResourceID,
			ErrorMsg:    errmsg,
		},
	}); err != nil {
		av.l.Fatal().Err(err).Str("uploadid", ev.UploadID).Interface("resourceID", ev.ResourceID).Msg("cannot publish events - exiting")
		return fmt.Errorf("%w: %s", ErrFatal, err)
	}
	return nil
}

// process the scan
func (av Antivirus) process(ev events.StartPostprocessingStep) (scanners.Result, error) {
	if ev.Filesize == 0 || (0 < av.m && av.m < ev.Filesize) {
		av.l.Info().Str("uploadid", ev.UploadID).Uint64("limit", av.m).Uint64("filesize", ev.Filesize).Msg("Skipping file to be virus scanned because its file size is higher than the defined limit.")
		return scanners.Result{
			ScanTime: time.Now(),
		}, nil
	}

	var err error
	var rrc io.ReadCloser

	switch ev.UploadID {
	default:
		rrc, err = av.downloadViaToken(ev.URL)
	case "":
		rrc, err = av.downloadViaReva(ev.URL, ev.Token, ev.RevaToken)
	}
	if err != nil {
		av.l.Error().Err(err).Str("uploadid", ev.UploadID).Msg("error downloading file")
		return scanners.Result{}, err
	}
	defer rrc.Close()
	av.l.Debug().Str("uploadid", ev.UploadID).Msg("Downloaded file successfully, starting virusscan")

	res, err := av.s.Scan(scanners.Input{Body: rrc, Size: int64(ev.Filesize), Url: ev.URL, Name: ev.Filename})
	if err != nil {
		av.l.Error().Err(err).Str("uploadid", ev.UploadID).Msg("error scanning file")
	}

	return res, err
}

// download will download the file
func (av Antivirus) downloadViaToken(url string) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	return av.doDownload(req)
}

// download will download the file
func (av Antivirus) downloadViaReva(url string, dltoken string, revatoken string) (io.ReadCloser, error) {
	ctx := ctxpkg.ContextSetToken(context.Background(), revatoken)

	req, err := rhttp.NewRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Reva-Transfer", dltoken)

	return av.doDownload(req)
}

func (av Antivirus) doDownload(req *http.Request) (io.ReadCloser, error) {
	res, err := av.client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, fmt.Errorf("unexpected status code from Download %v", res.StatusCode)
	}

	return res.Body, nil
}
