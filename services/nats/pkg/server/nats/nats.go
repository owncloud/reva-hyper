package nats

import (
	"time"

	nserver "github.com/nats-io/nats-server/v2/server"
)

var NATSListenAndServeLoopTimer = 1 * time.Second

type NATSServer struct {
	server *nserver.Server
}

func NewNATSServer(logger nserver.Logger, opts ...NatsOption) (*NATSServer, error) {
	natsOpts := &nserver.Options{}

	for _, o := range opts {
		o(natsOpts)
	}

	// enable JetStream
	natsOpts.JetStream = true

	server, err := nserver.NewServer(natsOpts)
	if err != nil {
		return nil, err
	}

	server.SetLoggerV2(logger, true, true, false)

	return &NATSServer{
		server: server,
	}, nil
}

// ListenAndServe runs the NATSServer in a blocking way until the server is shutdown or an error occurs
func (n *NATSServer) ListenAndServe() (err error) {
	n.server.Start()           // it won't block
	n.server.WaitForShutdown() // block until the server is fully shutdown
	return nil
}

func (n *NATSServer) Shutdown() {
	n.server.Shutdown()
}
