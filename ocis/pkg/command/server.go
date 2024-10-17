package command

import (
	"github.com/owncloud/ocis/v2/ocis-pkg/config"
	"github.com/owncloud/ocis/v2/ocis-pkg/config/configlog"
	"github.com/owncloud/ocis/v2/ocis-pkg/config/parser"
	"github.com/owncloud/ocis/v2/ocis/pkg/register"
	"github.com/owncloud/ocis/v2/ocis/pkg/runtime"
	"github.com/urfave/cli/v2"
)

// Server is the entrypoint for the server command.
func Server(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "server",
		Usage:    "start a fullstack server (runtime and all services in supervised mode)",
		Category: "fullstack",
		Before: func(c *cli.Context) error {
			return configlog.ReturnError(parser.ParseConfig(cfg, false))
		},
		Action: func(c *cli.Context) error {
			// Prefer the in-memory registry as the default when running in single-binary mode
			r := runtime.New(cfg)
			return r.Start(c.Context)
		},
	}
}

func init() {
	register.AddCommand(Server)
}
