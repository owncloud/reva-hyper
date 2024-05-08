package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	ctxpkg "github.com/cs3org/reva/v2/pkg/ctx"
	"github.com/cs3org/reva/v2/pkg/events"
	"github.com/cs3org/reva/v2/pkg/utils"
	"github.com/owncloud/ocis/v2/ocis-pkg/log"
	"github.com/owncloud/ocis/v2/services/postprocessing/pkg/config"
	"github.com/owncloud/ocis/v2/services/postprocessing/pkg/postprocessing"
	"go-micro.dev/v4/store"
	"go.opentelemetry.io/otel/trace"
)

// PostprocessingService is an instance of the service handling postprocessing of files
type PostprocessingService struct {
	ctx     context.Context
	log     log.Logger
	events  <-chan events.Event
	pub     events.Publisher
	steps   []events.Postprocessingstep
	store   store.Store
	c       config.Postprocessing
	tp      trace.TracerProvider
	stopCh  chan struct{}
	stopped atomic.Bool
}

var (
	// ErrFatal is returned when a fatal error occurs and we want to exit.
	ErrFatal = errors.New("fatal error")
	// ErrEvent is returned when something went wrong with a specific event.
	ErrEvent = errors.New("event error")
	// ErrNotFound is returned when a postprocessing is not found in the store.
	ErrNotFound = errors.New("postprocessing not found")
)

// NewPostprocessingService returns a new instance of a postprocessing service
func NewPostprocessingService(ctx context.Context, stream events.Stream, logger log.Logger, sto store.Store, tp trace.TracerProvider, c config.Postprocessing) (*PostprocessingService, error) {
	evs, err := events.Consume(stream, "postprocessing",
		events.BytesReceived{},
		events.StartPostprocessingStep{},
		events.UploadReady{},
		events.PostprocessingStepFinished{},
		events.ResumePostprocessing{},
	)
	if err != nil {
		return nil, err
	}

	return &PostprocessingService{
		ctx:    ctx,
		log:    logger,
		events: evs,
		pub:    stream,
		steps:  getSteps(c),
		store:  sto,
		c:      c,
		tp:     tp,
		stopCh: make(chan struct{}, 1),
	}, nil
}

// Run to fulfil Runner interface
func (pps *PostprocessingService) Run() error {
EventLoop:
	for {
		select {
		case <-pps.stopCh:
			// stop requested
			// TODO: we might need a way to unsubscribe from the event channel, otherwise
			// we'll be leaking a goroutine in reva that will be stuck waiting for
			// someone to read from the event channel.
			// Note: redis implementation seems to have a timeout, so the goroutine
			// will exit if there is nobody processing the events and the timeout
			// is reached. The behavior is unclear with natsjs
			break EventLoop
		case e, ok := <-pps.events:
			if !ok {
				// event channel is closed, so nothing more to do
				break EventLoop
			}

			err := pps.processEvent(e)
			if err != nil {
				switch {
				case errors.Is(err, ErrFatal):
					return err
				case errors.Is(err, ErrEvent):
					continue
				default:
					pps.log.Fatal().Err(err).Msg("unknown error - exiting")
				}
			}

			if pps.stopped.Load() {
				// if stopped, don't process any more events
				break EventLoop
			}
		}
	}
	return nil
}

// Close will make the postprocessing service to stop processing, so the `Run`
// method can finish.
// TODO: Underlying services can't be stopped. This means that some goroutines
// will get stuck trying to push events through a channel nobody is reading
// from, so resources won't be freed and there will be memory leaks. For now,
// if the service is stopped, you should close the app soon after.
func (pps *PostprocessingService) Close() {
	pps.stopped.Store(true)
	close(pps.stopCh)
}

func (pps *PostprocessingService) processEvent(e events.Event) error {
	var (
		next interface{}
		pp   *postprocessing.Postprocessing
		err  error
	)

	ctx := e.GetTraceContext(pps.ctx)
	ctx, span := pps.tp.Tracer("postprocessing").Start(ctx, "processEvent")
	defer span.End()

	switch ev := e.Event.(type) {
	case events.BytesReceived:
		pp = &postprocessing.Postprocessing{
			ID:          ev.UploadID,
			URL:         ev.URL,
			User:        ev.ExecutingUser,
			Filename:    ev.Filename,
			Filesize:    ev.Filesize,
			ResourceID:  ev.ResourceID,
			Steps:       pps.steps,
			InitiatorID: e.InitiatorID,
		}
		next = pp.Init(ev)
	case events.PostprocessingStepFinished:
		if ev.UploadID == "" {
			// no current upload - this was an on demand scan
			return nil
		}
		pp, err = pps.getPP(pps.store, ev.UploadID)
		if err != nil {
			pps.log.Error().Str("uploadID", ev.UploadID).Err(err).Msg("cannot get upload")
			return fmt.Errorf("%w: cannot get upload", ErrEvent)
		}
		next = pp.NextStep(ev)

		switch pp.Status.Outcome {
		case events.PPOutcomeRetry:
			// schedule retry
			backoff := pp.BackoffDuration()
			go func() {
				time.Sleep(backoff)
				retryEvent := events.StartPostprocessingStep{
					UploadID:      pp.ID,
					URL:           pp.URL,
					ExecutingUser: pp.User,
					Filename:      pp.Filename,
					Filesize:      pp.Filesize,
					ResourceID:    pp.ResourceID,
					StepToStart:   pp.Status.CurrentStep,
				}
				err := events.Publish(ctx, pps.pub, retryEvent)
				if err != nil {
					pps.log.Error().Str("uploadID", ev.UploadID).Err(err).Msg("cannot publish RestartPostprocessing event")
				}
			}()
		}
	case events.StartPostprocessingStep:
		if ev.StepToStart != events.PPStepDelay {
			return nil
		}
		pp, err = pps.getPP(pps.store, ev.UploadID)
		if err != nil {
			pps.log.Error().Str("uploadID", ev.UploadID).Err(err).Msg("cannot get upload")
			return fmt.Errorf("%w: cannot get upload", ErrEvent)
		}
		next = pp.Delay()
	case events.UploadReady:
		if ev.Failed {
			// the upload failed - let's keep it around for a while - but mark it as finished
			pp, err = pps.getPP(pps.store, ev.UploadID)
			if err != nil {
				pps.log.Error().Str("uploadID", ev.UploadID).Err(err).Msg("cannot get upload")
				return fmt.Errorf("%w: cannot get upload", ErrEvent)
			}
			pp.Finished = true
			return storePP(pps.store, pp)
		}

		// the storage provider thinks the upload is done - so no need to keep it any more
		if err := pps.store.Delete(ev.UploadID); err != nil {
			pps.log.Error().Str("uploadID", ev.UploadID).Err(err).Msg("cannot delete upload")
			return fmt.Errorf("%w: cannot delete upload", ErrEvent)
		}
	case events.ResumePostprocessing:
		return pps.handleResumePPEvent(ctx, ev)
	}

	if pp != nil {
		ctx = ctxpkg.ContextSetInitiator(ctx, pp.InitiatorID)

		if err := storePP(pps.store, pp); err != nil {
			pps.log.Error().Str("uploadID", pp.ID).Err(err).Msg("cannot store upload")
			return fmt.Errorf("%w: cannot store upload", ErrEvent)
		}
	}

	if next != nil {
		if err := events.Publish(ctx, pps.pub, next); err != nil {
			pps.log.Error().Err(err).Msg("unable to publish event")
			return fmt.Errorf("%w: unable to publish event", ErrFatal) // we can't publish -> we are screwed
		}
	}
	return nil
}

func (pps *PostprocessingService) getPP(sto store.Store, uploadID string) (*postprocessing.Postprocessing, error) {
	recs, err := sto.Read(uploadID)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if len(recs) == 0 {
		return nil, ErrNotFound
	}

	if len(recs) > 1 {
		return nil, fmt.Errorf("expected only one result for '%s', got %d", uploadID, len(recs))
	}

	pp := postprocessing.New(pps.c)
	err = json.Unmarshal(recs[0].Value, pp)
	if err != nil {
		return nil, err
	}

	return pp, nil
}

func getSteps(c config.Postprocessing) []events.Postprocessingstep {
	// NOTE: improved version only allows configuring order of postprocessing steps
	// But we aim for a system where postprocessing steps can be configured per space, ideally by the spaceadmin itself
	// We need to iterate over configuring PP service when we see fit
	steps := make([]events.Postprocessingstep, 0, len(c.Steps))
	for _, s := range c.Steps {
		steps = append(steps, events.Postprocessingstep(s))
	}

	return steps
}

func storePP(sto store.Store, pp *postprocessing.Postprocessing) error {
	b, err := json.Marshal(pp)
	if err != nil {
		return err
	}

	return sto.Write(&store.Record{
		Key:   pp.ID,
		Value: b,
	})
}

func (pps *PostprocessingService) handleResumePPEvent(ctx context.Context, ev events.ResumePostprocessing) error {
	ids := []string{ev.UploadID}
	if ev.Step != "" {
		ids = pps.findUploadsByStep(ev.Step)
	}

	for _, id := range ids {
		if err := pps.resumePP(ctx, id); err != nil {
			pps.log.Error().Str("uploadID", id).Err(err).Msg("cannot resume upload")
			return fmt.Errorf("cannot resume upload: %w", err)
		}
	}
	return nil
}

func (pps *PostprocessingService) resumePP(ctx context.Context, uploadID string) error {
	pp, err := pps.getPP(pps.store, uploadID)
	if err != nil {
		if err == ErrNotFound {
			if err := events.Publish(ctx, pps.pub, events.RestartPostprocessing{
				UploadID:  uploadID,
				Timestamp: utils.TSNow(),
			}); err != nil {
				return err
			}
			return nil
		}
		return fmt.Errorf("cannot get upload: %w", err)
	}

	if pp.Finished {
		// dont retry finished uploads
		return nil
	}

	return events.Publish(ctx, pps.pub, pp.CurrentStep())
}

func (pps *PostprocessingService) findUploadsByStep(step events.Postprocessingstep) []string {
	var ids []string

	keys, err := pps.store.List()
	if err != nil {
		pps.log.Error().Err(err).Msg("cannot list uploads")
	}

	for _, k := range keys {
		rec, err := pps.store.Read(k)
		if err != nil {
			pps.log.Error().Err(err).Msg("cannot read upload")
			continue
		}

		if len(rec) != 1 {
			pps.log.Error().Err(err).Msg("expected only one result")
			continue
		}

		pp := &postprocessing.Postprocessing{}
		err = json.Unmarshal(rec[0].Value, pp)
		if err != nil {
			pps.log.Error().Err(err).Msg("cannot unmarshal upload")
			continue
		}

		if pp.Status.CurrentStep == step {
			ids = append(ids, pp.ID)
		}
	}

	return ids
}
