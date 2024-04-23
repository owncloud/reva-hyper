package command

import (
	"context"
	"fmt"
	"github.com/DeepDiver1975/go-webdav/caldav"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	revaContext "github.com/cs3org/reva/v2/pkg/ctx"
	config2 "github.com/owncloud/ocis/v2/ocis-pkg/config"
	parser2 "github.com/owncloud/ocis/v2/ocis-pkg/config/parser"
	ogrpc "github.com/owncloud/ocis/v2/ocis-pkg/service/grpc"
	"github.com/owncloud/ocis/v2/services/ccs/pkg/config"
	"github.com/owncloud/ocis/v2/services/ccs/pkg/config/parser"
	svc "github.com/owncloud/ocis/v2/services/ccs/pkg/service/v0"
	"github.com/urfave/cli/v2"
)

func CreateCalendar(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "create-calendar",
		Usage:    "create calendar for user",
		Category: "maintenance",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "user-name",
				Value:    "string",
				Usage:    "user name",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "calendar-name",
				Value:    "string",
				Usage:    "calendar name",
				Required: true,
			},
		},
		Before: func(c *cli.Context) error {
			// load ocis config if possible
			ocisConfig := config2.Config{}
			err := parser2.ParseConfig(&ocisConfig, false)
			if err != nil {
				return err
			}
			cfg.Commons = ocisConfig.Commons
			err = parser.ParseConfig(cfg)
			if err != nil {
				return err
			}

			return nil
		},
		Action: func(c *cli.Context) error {
			// init grpc connection
			_, err := ogrpc.NewClient()
			if err != nil {
				return err
			}

			_, backend, _, err := svc.InitStorage(c.Context, cfg.Storage)
			if err != nil {
				return err
			}
			userName := c.String("user-name")
			calendarName := c.String("calendar-name")
			path := fmt.Sprintf("/dav/calendars/%s/%s", userName, calendarName)

			calendar := caldav.Calendar{
				Path: path,
				Name: calendarName,
			}
			u := userpb.User{
				Username: userName,
			}
			ctx := revaContext.ContextSetUser(context.Background(), &u)
			return backend.CreateCalendar(ctx, &calendar)
		},
	}
}
