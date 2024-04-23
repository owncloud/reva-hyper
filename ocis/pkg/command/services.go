package command

import (
	ccs "github.com/owncloud/ocis/v2/services/ccs/pkg/command"
	"github.com/urfave/cli/v2"

	"github.com/owncloud/ocis/v2/ocis-pkg/config"
	"github.com/owncloud/ocis/v2/ocis-pkg/config/configlog"
	"github.com/owncloud/ocis/v2/ocis-pkg/config/parser"
	"github.com/owncloud/ocis/v2/ocis/pkg/command/helper"
	"github.com/owncloud/ocis/v2/ocis/pkg/register"
	antivirus "github.com/owncloud/ocis/v2/services/antivirus/pkg/command"
	appprovider "github.com/owncloud/ocis/v2/services/app-provider/pkg/command"
	appregistry "github.com/owncloud/ocis/v2/services/app-registry/pkg/command"
	audit "github.com/owncloud/ocis/v2/services/audit/pkg/command"
	authbasic "github.com/owncloud/ocis/v2/services/auth-basic/pkg/command"
	authbearer "github.com/owncloud/ocis/v2/services/auth-bearer/pkg/command"
	authmachine "github.com/owncloud/ocis/v2/services/auth-machine/pkg/command"
	authservice "github.com/owncloud/ocis/v2/services/auth-service/pkg/command"
	clientlog "github.com/owncloud/ocis/v2/services/clientlog/pkg/command"
	eventhistory "github.com/owncloud/ocis/v2/services/eventhistory/pkg/command"
	frontend "github.com/owncloud/ocis/v2/services/frontend/pkg/command"
	gateway "github.com/owncloud/ocis/v2/services/gateway/pkg/command"
	graph "github.com/owncloud/ocis/v2/services/graph/pkg/command"
	groups "github.com/owncloud/ocis/v2/services/groups/pkg/command"
	idm "github.com/owncloud/ocis/v2/services/idm/pkg/command"
	idp "github.com/owncloud/ocis/v2/services/idp/pkg/command"
	invitations "github.com/owncloud/ocis/v2/services/invitations/pkg/command"
	nats "github.com/owncloud/ocis/v2/services/nats/pkg/command"
	notifications "github.com/owncloud/ocis/v2/services/notifications/pkg/command"
	ocdav "github.com/owncloud/ocis/v2/services/ocdav/pkg/command"
	ocm "github.com/owncloud/ocis/v2/services/ocm/pkg/command"
	ocs "github.com/owncloud/ocis/v2/services/ocs/pkg/command"
	policies "github.com/owncloud/ocis/v2/services/policies/pkg/command"
	postprocessing "github.com/owncloud/ocis/v2/services/postprocessing/pkg/command"
	proxy "github.com/owncloud/ocis/v2/services/proxy/pkg/command"
	search "github.com/owncloud/ocis/v2/services/search/pkg/command"
	settings "github.com/owncloud/ocis/v2/services/settings/pkg/command"
	sharing "github.com/owncloud/ocis/v2/services/sharing/pkg/command"
	sse "github.com/owncloud/ocis/v2/services/sse/pkg/command"
	storagepubliclink "github.com/owncloud/ocis/v2/services/storage-publiclink/pkg/command"
	storageshares "github.com/owncloud/ocis/v2/services/storage-shares/pkg/command"
	storagesystem "github.com/owncloud/ocis/v2/services/storage-system/pkg/command"
	storageusers "github.com/owncloud/ocis/v2/services/storage-users/pkg/command"
	store "github.com/owncloud/ocis/v2/services/store/pkg/command"
	thumbnails "github.com/owncloud/ocis/v2/services/thumbnails/pkg/command"
	userlog "github.com/owncloud/ocis/v2/services/userlog/pkg/command"
	users "github.com/owncloud/ocis/v2/services/users/pkg/command"
	web "github.com/owncloud/ocis/v2/services/web/pkg/command"
	webdav "github.com/owncloud/ocis/v2/services/webdav/pkg/command"
	webfinger "github.com/owncloud/ocis/v2/services/webfinger/pkg/command"
)

var svccmds = []register.Command{
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Antivirus.Service.Name, antivirus.GetCommands(cfg.Antivirus), func(c *config.Config) {
			// cfg.Antivirus.Commons = cfg.Commons // antivirus needs no commons atm
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.AppProvider.Service.Name, appprovider.GetCommands(cfg.AppProvider), func(c *config.Config) {
			cfg.AppProvider.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.AppRegistry.Service.Name, appregistry.GetCommands(cfg.AppRegistry), func(c *config.Config) {
			cfg.AppRegistry.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Audit.Service.Name, audit.GetCommands(cfg.Audit), func(c *config.Config) {
			cfg.Audit.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.AuthBasic.Service.Name, authbasic.GetCommands(cfg.AuthBasic), func(c *config.Config) {
			cfg.AuthBasic.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.AuthBearer.Service.Name, authbearer.GetCommands(cfg.AuthBearer), func(c *config.Config) {
			cfg.AuthBearer.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.AuthMachine.Service.Name, authmachine.GetCommands(cfg.AuthMachine), func(c *config.Config) {
			cfg.AuthMachine.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.AuthService.Service.Name, authservice.GetCommands(cfg.AuthService), func(c *config.Config) {
			cfg.AuthService.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.CCS.Service.Name, ccs.GetCommands(cfg.CCS), func(c *config.Config) {
			cfg.Clientlog.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Clientlog.Service.Name, clientlog.GetCommands(cfg.Clientlog), func(c *config.Config) {
			cfg.Clientlog.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.EventHistory.Service.Name, eventhistory.GetCommands(cfg.EventHistory), func(c *config.Config) {
			cfg.EventHistory.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Frontend.Service.Name, frontend.GetCommands(cfg.Frontend), func(c *config.Config) {
			cfg.Frontend.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Gateway.Service.Name, gateway.GetCommands(cfg.Gateway), func(c *config.Config) {
			cfg.Gateway.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Graph.Service.Name, graph.GetCommands(cfg.Graph), func(c *config.Config) {
			cfg.Graph.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Groups.Service.Name, groups.GetCommands(cfg.Groups), func(c *config.Config) {
			cfg.Groups.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.IDM.Service.Name, idm.GetCommands(cfg.IDM), func(c *config.Config) {
			cfg.IDM.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.IDP.Service.Name, idp.GetCommands(cfg.IDP), func(c *config.Config) {
			cfg.IDP.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Invitations.Service.Name, invitations.GetCommands(cfg.Invitations), func(c *config.Config) {
			cfg.Invitations.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Nats.Service.Name, nats.GetCommands(cfg.Nats), func(c *config.Config) {
			cfg.Nats.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Notifications.Service.Name, notifications.GetCommands(cfg.Notifications), func(c *config.Config) {
			cfg.Notifications.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.OCDav.Service.Name, ocdav.GetCommands(cfg.OCDav), func(c *config.Config) {
			cfg.OCDav.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.OCM.Service.Name, ocm.GetCommands(cfg.OCM), func(c *config.Config) {
			cfg.OCM.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.OCS.Service.Name, ocs.GetCommands(cfg.OCS), func(c *config.Config) {
			cfg.OCS.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Policies.Service.Name, policies.GetCommands(cfg.Policies), func(c *config.Config) {
			cfg.Policies.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Postprocessing.Service.Name, postprocessing.GetCommands(cfg.Postprocessing), func(c *config.Config) {
			cfg.Postprocessing.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Proxy.Service.Name, proxy.GetCommands(cfg.Proxy), func(c *config.Config) {
			cfg.Proxy.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Search.Service.Name, search.GetCommands(cfg.Search), func(c *config.Config) {
			cfg.Search.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Settings.Service.Name, settings.GetCommands(cfg.Settings), func(c *config.Config) {
			cfg.Settings.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Sharing.Service.Name, sharing.GetCommands(cfg.Sharing), func(c *config.Config) {
			cfg.Sharing.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.SSE.Service.Name, sse.GetCommands(cfg.SSE), func(c *config.Config) {
			cfg.SSE.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.StoragePublicLink.Service.Name, storagepubliclink.GetCommands(cfg.StoragePublicLink), func(c *config.Config) {
			cfg.StoragePublicLink.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.StorageShares.Service.Name, storageshares.GetCommands(cfg.StorageShares), func(c *config.Config) {
			cfg.StorageShares.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.StorageSystem.Service.Name, storagesystem.GetCommands(cfg.StorageSystem), func(c *config.Config) {
			cfg.StorageSystem.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.StorageUsers.Service.Name, storageusers.GetCommands(cfg.StorageUsers), func(c *config.Config) {
			cfg.StorageUsers.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Store.Service.Name, store.GetCommands(cfg.Store), func(c *config.Config) {
			cfg.Store.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Thumbnails.Service.Name, thumbnails.GetCommands(cfg.Thumbnails), func(c *config.Config) {
			cfg.Thumbnails.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Userlog.Service.Name, userlog.GetCommands(cfg.Userlog), func(c *config.Config) {
			cfg.Userlog.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Users.Service.Name, users.GetCommands(cfg.Users), func(c *config.Config) {
			cfg.Users.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Web.Service.Name, web.GetCommands(cfg.Web), func(c *config.Config) {
			cfg.Web.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.WebDAV.Service.Name, webdav.GetCommands(cfg.WebDAV), func(c *config.Config) {
			cfg.WebDAV.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cli.Command {
		return ServiceCommand(cfg, cfg.Webfinger.Service.Name, webfinger.GetCommands(cfg.Webfinger), func(c *config.Config) {
			cfg.Webfinger.Commons = cfg.Commons
		})
	},
}

// ServiceCommand is the entry point for the all service commands.
func ServiceCommand(cfg *config.Config, servicename string, subcommands []*cli.Command, f func(*config.Config)) *cli.Command {
	return &cli.Command{
		Name:     servicename,
		Usage:    helper.SubcommandDescription(servicename),
		Category: "services",
		Before: func(c *cli.Context) error {
			configlog.Error(parser.ParseConfig(cfg, true))
			f(cfg)
			return nil
		},
		Subcommands: subcommands,
	}
}

func init() {
	for _, c := range svccmds {
		register.AddCommand(c)
	}
}
