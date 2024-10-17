package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/owncloud/ocis/v2/services/sse/pkg/command"
	"github.com/owncloud/ocis/v2/services/sse/pkg/config/defaults"
)

func main() {
	cfg := defaults.DefaultConfig()
	cfg.Context, _ = signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	if err := command.Execute(cfg); err != nil {
		os.Exit(1)
	}
}
