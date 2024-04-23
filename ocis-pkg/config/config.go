package config

import (
	"github.com/owncloud/ocis/v2/ocis-pkg/shared"
	antivirus "github.com/owncloud/ocis/v2/services/antivirus/pkg/config"
	appProvider "github.com/owncloud/ocis/v2/services/app-provider/pkg/config"
	appRegistry "github.com/owncloud/ocis/v2/services/app-registry/pkg/config"
	audit "github.com/owncloud/ocis/v2/services/audit/pkg/config"
	authbasic "github.com/owncloud/ocis/v2/services/auth-basic/pkg/config"
	authbearer "github.com/owncloud/ocis/v2/services/auth-bearer/pkg/config"
	authmachine "github.com/owncloud/ocis/v2/services/auth-machine/pkg/config"
	authservice "github.com/owncloud/ocis/v2/services/auth-service/pkg/config"
	ccs "github.com/owncloud/ocis/v2/services/ccs/pkg/config"
	clientlog "github.com/owncloud/ocis/v2/services/clientlog/pkg/config"
	eventhistory "github.com/owncloud/ocis/v2/services/eventhistory/pkg/config"
	frontend "github.com/owncloud/ocis/v2/services/frontend/pkg/config"
	gateway "github.com/owncloud/ocis/v2/services/gateway/pkg/config"
	graph "github.com/owncloud/ocis/v2/services/graph/pkg/config"
	groups "github.com/owncloud/ocis/v2/services/groups/pkg/config"
	idm "github.com/owncloud/ocis/v2/services/idm/pkg/config"
	idp "github.com/owncloud/ocis/v2/services/idp/pkg/config"
	invitations "github.com/owncloud/ocis/v2/services/invitations/pkg/config"
	nats "github.com/owncloud/ocis/v2/services/nats/pkg/config"
	notifications "github.com/owncloud/ocis/v2/services/notifications/pkg/config"
	ocdav "github.com/owncloud/ocis/v2/services/ocdav/pkg/config"
	ocm "github.com/owncloud/ocis/v2/services/ocm/pkg/config"
	ocs "github.com/owncloud/ocis/v2/services/ocs/pkg/config"
	policies "github.com/owncloud/ocis/v2/services/policies/pkg/config"
	postprocessing "github.com/owncloud/ocis/v2/services/postprocessing/pkg/config"
	proxy "github.com/owncloud/ocis/v2/services/proxy/pkg/config"
	search "github.com/owncloud/ocis/v2/services/search/pkg/config"
	settings "github.com/owncloud/ocis/v2/services/settings/pkg/config"
	sharing "github.com/owncloud/ocis/v2/services/sharing/pkg/config"
	sse "github.com/owncloud/ocis/v2/services/sse/pkg/config"
	storagepublic "github.com/owncloud/ocis/v2/services/storage-publiclink/pkg/config"
	storageshares "github.com/owncloud/ocis/v2/services/storage-shares/pkg/config"
	storagesystem "github.com/owncloud/ocis/v2/services/storage-system/pkg/config"
	storageusers "github.com/owncloud/ocis/v2/services/storage-users/pkg/config"
	store "github.com/owncloud/ocis/v2/services/store/pkg/config"
	thumbnails "github.com/owncloud/ocis/v2/services/thumbnails/pkg/config"
	userlog "github.com/owncloud/ocis/v2/services/userlog/pkg/config"
	users "github.com/owncloud/ocis/v2/services/users/pkg/config"
	web "github.com/owncloud/ocis/v2/services/web/pkg/config"
	webdav "github.com/owncloud/ocis/v2/services/webdav/pkg/config"
	webfinger "github.com/owncloud/ocis/v2/services/webfinger/pkg/config"
)

type Mode int

// Runtime configures the oCIS runtime when running in supervised mode.
type Runtime struct {
	Port       string   `yaml:"port" env:"OCIS_RUNTIME_PORT" desc:"The TCP port at which oCIS will be available" introductionVersion:"pre5.0"`
	Host       string   `yaml:"host" env:"OCIS_RUNTIME_HOST" desc:"The host at which oCIS will be available" introductionVersion:"pre5.0"`
	Services   []string `yaml:"services" env:"OCIS_RUN_EXTENSIONS;OCIS_RUN_SERVICES" desc:"A comma-separated list of service names. Will start only the listed services." introductionVersion:"pre5.0"`
	Disabled   []string `yaml:"disabled_services" env:"OCIS_EXCLUDE_RUN_SERVICES" desc:"A comma-separated list of service names. Will start all default services except of the ones listed. Has no effect when OCIS_RUN_SERVICES is set." introductionVersion:"pre5.0"`
	Additional []string `yaml:"add_services" env:"OCIS_ADD_RUN_SERVICES" desc:"A comma-separated list of service names. Will add the listed services to the default configuration. Has no effect when OCIS_RUN_SERVICES is set. Note that one can add services not started by the default list and exclude services from the default list by using both envvars at the same time." introductionVersion:"pre5.0"`
}

// Config combines all available configuration parts.
type Config struct {
	*shared.Commons `yaml:"shared"`

	Tracing        *shared.Tracing        `yaml:"tracing"`
	Log            *shared.Log            `yaml:"log"`
	Cache          *shared.Cache          `yaml:"cache"`
	GRPCClientTLS  *shared.GRPCClientTLS  `yaml:"grpc_client_tls"`
	GRPCServiceTLS *shared.GRPCServiceTLS `yaml:"grpc_service_tls"`
	HTTPServiceTLS shared.HTTPServiceTLS  `yaml:"http_service_tls"`

	Mode    Mode // DEPRECATED
	File    string
	OcisURL string `yaml:"ocis_url" env:"OCIS_URL" desc:"URL, where oCIS is reachable for users." introductionVersion:"pre5.0"`

	Registry          string               `yaml:"registry"`
	TokenManager      *shared.TokenManager `yaml:"token_manager"`
	MachineAuthAPIKey string               `yaml:"machine_auth_api_key" env:"OCIS_MACHINE_AUTH_API_KEY" desc:"Machine auth API key used to validate internal requests necessary for the access to resources from other services." introductionVersion:"pre5.0"`
	TransferSecret    string               `yaml:"transfer_secret" env:"OCIS_TRANSFER_SECRET" desc:"Transfer secret for signing file up- and download requests." introductionVersion:"pre5.0"`
	SystemUserID      string               `yaml:"system_user_id" env:"OCIS_SYSTEM_USER_ID" desc:"ID of the oCIS storage-system system user. Admins need to set the ID for the storage-system system user in this config option which is then used to reference the user. Any reasonable long string is possible, preferably this would be an UUIDv4 format." introductionVersion:"pre5.0"`
	SystemUserAPIKey  string               `yaml:"system_user_api_key" env:"OCIS_SYSTEM_USER_API_KEY" desc:"API key for the storage-system system user." introductionVersion:"pre5.0"`
	AdminUserID       string               `yaml:"admin_user_id" env:"OCIS_ADMIN_USER_ID" desc:"ID of a user, that should receive admin privileges. Consider that the UUID can be encoded in some LDAP deployment configurations like in .ldif files. These need to be decoded beforehand." introductionVersion:"pre5.0"`
	Runtime           Runtime              `yaml:"runtime"`

	Antivirus         *antivirus.Config      `yaml:"antivirus"`
	AppProvider       *appProvider.Config    `yaml:"app_provider"`
	AppRegistry       *appRegistry.Config    `yaml:"app_registry"`
	Audit             *audit.Config          `yaml:"audit"`
	AuthBasic         *authbasic.Config      `yaml:"auth_basic"`
	AuthBearer        *authbearer.Config     `yaml:"auth_bearer"`
	AuthMachine       *authmachine.Config    `yaml:"auth_machine"`
	AuthService       *authservice.Config    `yaml:"auth_service"`
	CCS               *ccs.Config            `yaml:"ccs"`
	Clientlog         *clientlog.Config      `yaml:"clientlog"`
	EventHistory      *eventhistory.Config   `yaml:"eventhistory"`
	Frontend          *frontend.Config       `yaml:"frontend"`
	Gateway           *gateway.Config        `yaml:"gateway"`
	Graph             *graph.Config          `yaml:"graph"`
	Groups            *groups.Config         `yaml:"groups"`
	IDM               *idm.Config            `yaml:"idm"`
	IDP               *idp.Config            `yaml:"idp"`
	Invitations       *invitations.Config    `yaml:"invitations"`
	Nats              *nats.Config           `yaml:"nats"`
	Notifications     *notifications.Config  `yaml:"notifications"`
	OCDav             *ocdav.Config          `yaml:"ocdav"`
	OCM               *ocm.Config            `yaml:"ocm"`
	OCS               *ocs.Config            `yaml:"ocs"`
	Postprocessing    *postprocessing.Config `yaml:"postprocessing"`
	Policies          *policies.Config       `yaml:"policies"`
	Proxy             *proxy.Config          `yaml:"proxy"`
	Settings          *settings.Config       `yaml:"settings"`
	Sharing           *sharing.Config        `yaml:"sharing"`
	SSE               *sse.Config            `yaml:"sse"`
	StorageSystem     *storagesystem.Config  `yaml:"storage_system"`
	StoragePublicLink *storagepublic.Config  `yaml:"storage_public"`
	StorageShares     *storageshares.Config  `yaml:"storage_shares"`
	StorageUsers      *storageusers.Config   `yaml:"storage_users"`
	Store             *store.Config          `yaml:"store"`
	Thumbnails        *thumbnails.Config     `yaml:"thumbnails"`
	Userlog           *userlog.Config        `yaml:"userlog"`
	Users             *users.Config          `yaml:"users"`
	Web               *web.Config            `yaml:"web"`
	WebDAV            *webdav.Config         `yaml:"webdav"`
	Webfinger         *webfinger.Config      `yaml:"webfinger"`
	Search            *search.Config         `yaml:"search"`
}
