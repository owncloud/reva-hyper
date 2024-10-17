package natsjsregistry

import (
	"encoding/json"
	"errors"
	"strings"

	natsjskv "github.com/go-micro/plugins/v4/store/nats-js-kv"
	"github.com/nats-io/nats.go"
	"go-micro.dev/v4/registry"
)

// NatsWatcher is the watcher of the nats interface
type NatsWatcher interface {
	WatchAll(bucket string, opts ...nats.WatchOpt) (<-chan *natsjskv.StoreUpdate, func() error, error)
}

// Watcher is used to keep track of changes in the registry
type Watcher struct {
	updates <-chan *natsjskv.StoreUpdate
	stop    func() error
	reg     *storeregistry
}

// NewWatcher returns a new watcher
func NewWatcher(s *storeregistry) (*Watcher, error) {
	w, ok := s.store.(NatsWatcher)
	if !ok {
		return nil, errors.New("store does not implement watcher interface")
	}

	watcher, stop, err := w.WatchAll("service-registry")
	if err != nil {
		return nil, err
	}

	return &Watcher{
		updates: watcher,
		stop:    stop,
		reg:     s,
	}, nil
}

// Next returns the next result. It is a blocking call
func (w *Watcher) Next() (*registry.Result, error) {
	kve := <-w.updates
	if kve == nil {
		return nil, errors.New("watcher stopped")
	}

	var svc registry.Service
	if kve.Value.Data == nil {
		// fake a service
		parts := strings.SplitN(kve.Value.Key, _serviceDelimiter, 3)
		if len(parts) != 3 {
			return nil, errors.New("invalid service key")
		}
		svc.Name = parts[0]
		// ocis registers nodes with a - separator
		svc.Nodes = []*registry.Node{{Id: parts[0] + "-" + parts[1]}}
		svc.Version = parts[2]
	} else {
		if err := json.Unmarshal(kve.Value.Data, &svc); err != nil {
			_ = w.stop()
			return nil, err
		}
	}

	return &registry.Result{
		Service: &svc,
		Action:  kve.Action,
	}, nil
}

// Stop stops the watcher
func (w *Watcher) Stop() {
	_ = w.stop()
}
