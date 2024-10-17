package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	cs3 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	revactx "github.com/cs3org/reva/v2/pkg/ctx"
	"github.com/cs3org/reva/v2/pkg/rgrpc/todo/pool"
	libregraph "github.com/owncloud/libre-graph-api-go"
	"github.com/owncloud/ocis/v2/ocis-pkg/log"
	"github.com/owncloud/ocis/v2/ocis-pkg/oidc"
	"github.com/owncloud/ocis/v2/services/graph/pkg/errorcode"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/config"
	"go-micro.dev/v4/selector"
)

type cs3backend struct {
	graphSelector selector.Selector
	Options
}

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	logger            log.Logger
	gatewaySelector   pool.Selectable[gateway.GatewayAPIClient]
	selector          selector.Selector
	machineAuthAPIKey string
	oidcISS           string
	serviceAccount    config.ServiceAccount
}

// WithLogger sets the logger option
func WithLogger(l log.Logger) Option {
	return func(o *Options) {
		o.logger = l
	}
}

// WithRevaGatewaySelector set the gatewaySelector option
func WithRevaGatewaySelector(selectable pool.Selectable[gateway.GatewayAPIClient]) Option {
	return func(o *Options) {
		o.gatewaySelector = selectable
	}
}

// WithSelector set the Selector option
func WithSelector(selector selector.Selector) Option {
	return func(o *Options) {
		o.selector = selector
	}
}

// WithMachineAuthAPIKey configures the machine auth API key
func WithMachineAuthAPIKey(ma string) Option {
	return func(o *Options) {
		o.machineAuthAPIKey = ma
	}
}

// WithOIDCissuer set the OIDC issuer URL
func WithOIDCissuer(oidcISS string) Option {
	return func(o *Options) {
		o.oidcISS = oidcISS
	}
}

// WithServiceAccount configures the service account creator to use
func WithServiceAccount(c config.ServiceAccount) Option {
	return func(o *Options) {
		o.serviceAccount = c
	}
}

// NewCS3UserBackend creates a user-provider which fetches users from a CS3 UserBackend
func NewCS3UserBackend(opts ...Option) UserBackend {
	opt := Options{}
	for _, o := range opts {
		o(&opt)
	}

	b := cs3backend{
		Options:       opt,
		graphSelector: opt.selector,
	}

	return &b
}

func (c *cs3backend) GetUserByClaims(ctx context.Context, claim, value string) (*cs3.User, string, error) {
	gatewayClient, err := c.gatewaySelector.Next()
	if err != nil {
		return nil, "", fmt.Errorf("could not obtain gatewayClient: %s", err)
	}

	res, err := gatewayClient.Authenticate(ctx, &gateway.AuthenticateRequest{
		Type:         "machine",
		ClientId:     claim + ":" + value,
		ClientSecret: c.machineAuthAPIKey,
	})

	switch {
	case err != nil:
		return nil, "", fmt.Errorf("could not get user by claim %v with value %v: %w", claim, value, err)
	case res.Status.Code != rpcv1beta1.Code_CODE_OK:
		if res.Status.Code == rpcv1beta1.Code_CODE_NOT_FOUND {
			return nil, "", ErrAccountNotFound
		}
		return nil, "", fmt.Errorf("could not get user by claim %v with value %v : %s ", claim, value, res.Status.Message)
	}

	user := res.User

	return user, res.Token, nil
}

func (c *cs3backend) Authenticate(ctx context.Context, username string, password string) (*cs3.User, string, error) {
	gatewayClient, err := c.gatewaySelector.Next()
	if err != nil {
		return nil, "", fmt.Errorf("could not obtain gatewayClient: %s", err)
	}

	res, err := gatewayClient.Authenticate(ctx, &gateway.AuthenticateRequest{
		Type:         "basic",
		ClientId:     username,
		ClientSecret: password,
	})

	switch {
	case err != nil:
		return nil, "", fmt.Errorf("could not authenticate with username and password user: %s, %w", username, err)
	case res.Status.Code != rpcv1beta1.Code_CODE_OK:
		return nil, "", fmt.Errorf("could not authenticate with username and password user: %s, got code: %d", username, res.Status.Code)
	}

	return res.User, res.Token, nil
}

// CreateUserFromClaims creates a new user via libregraph users API, taking the
// attributes from the provided `claims` map. On success it returns the new
// user. If the user already exist this is not considered an error and the
// function will just return the existing user.
func (c *cs3backend) CreateUserFromClaims(ctx context.Context, claims map[string]interface{}) (*cs3.User, error) {
	gatewayClient, err := c.gatewaySelector.Next()
	if err != nil {
		c.logger.Error().Err(err).Msg("could not select next gateway client")
		return nil, err
	}
	newctx := context.Background()
	authRes, err := gatewayClient.Authenticate(newctx, &gateway.AuthenticateRequest{
		Type:         "serviceaccounts",
		ClientId:     c.serviceAccount.ServiceAccountID,
		ClientSecret: c.serviceAccount.ServiceAccountSecret,
	})
	if err != nil {
		return nil, err
	}
	if authRes.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
		return nil, fmt.Errorf("error authenticating service user: %s", authRes.Status.Message)
	}

	lgClient, err := c.setupLibregraphClient(newctx, authRes.Token)
	if err != nil {
		c.logger.Error().Err(err).Msg("Error setting up libregraph client.")
		return nil, err
	}

	newUser, err := c.libregraphUserFromClaims(newctx, claims)
	if err != nil {
		c.logger.Error().Err(err).Interface("claims", claims).Msg("Error creating user from claims")
		return nil, fmt.Errorf("Error creating user from claims: %w", err)
	}

	req := lgClient.UsersApi.CreateUser(newctx).User(newUser)

	created, resp, err := req.Execute()
	defer resp.Body.Close()

	var reread bool
	if err != nil {
		if resp == nil {
			return nil, err
		}

		// If the user already exists here, some other request did already create it in parallel.
		// So just issue a Debug message and ignore the libregraph error otherwise
		var lerr error
		if reread, lerr = c.isAlreadyExists(resp); lerr != nil {
			c.logger.Error().Err(lerr).Msg("extracting error from ibregraph response body failed.")
			return nil, err
		}
		if !reread {
			c.logger.Error().Err(err).Msg("Error creating user")
			return nil, err
		}
	}

	// User has been created meanwhile, re-read it to get the user id
	if reread {
		c.logger.Debug().Msg("User already exist, re-reading via libregraph")
		gureq := lgClient.UserApi.GetUser(newctx, newUser.GetOnPremisesSamAccountName())
		created, resp, err = gureq.Execute()
		defer resp.Body.Close()
		if err != nil {
			c.logger.Error().Err(err).Msg("Error trying to re-read user from graphAPI")
			return nil, err
		}
	}

	cs3UserCreated := c.cs3UserFromLibregraph(newctx, created)

	return &cs3UserCreated, nil
}

func (c cs3backend) setupLibregraphClient(ctx context.Context, cs3token string) (*libregraph.APIClient, error) {
	// Use micro registry to resolve next graph service endpoint
	next, err := c.graphSelector.Select("com.owncloud.graph.graph")
	if err != nil {
		c.logger.Debug().Err(err).Msg("setupLibregraphClient: error during Select")
		return nil, err
	}
	node, err := next()
	if err != nil {
		c.logger.Debug().Err(err).Msg("setupLibregraphClient: error getting next Node")
		return nil, err
	}
	lgconf := libregraph.NewConfiguration()
	lgconf.Servers = libregraph.ServerConfigurations{
		{
			URL: fmt.Sprintf("%s://%s/graph", node.Metadata["protocol"], node.Address),
		},
	}

	lgconf.DefaultHeader = map[string]string{revactx.TokenHeader: cs3token}
	return libregraph.NewAPIClient(lgconf), nil
}

func (c cs3backend) isAlreadyExists(resp *http.Response) (bool, error) {
	oDataErr := libregraph.NewOdataErrorWithDefaults()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Debug().Err(err).Msg("Error trying to read libregraph response")
		return false, err
	}
	err = json.Unmarshal(body, oDataErr)
	if err != nil {
		c.logger.Debug().Err(err).Msg("Error unmarshalling libregraph response")
		return false, err
	}

	c.logger.Warn().Str("OData Error", oDataErr.Error.Message).Msg("Error Response")

	if oDataErr.Error.Code == errorcode.NameAlreadyExists.String() {
		return true, nil
	}
	return false, nil
}

func (c cs3backend) libregraphUserFromClaims(ctx context.Context, claims map[string]interface{}) (libregraph.User, error) {
	var ok bool
	var dn, mail, username string
	user := libregraph.User{}
	if dn, ok = claims[oidc.Name].(string); !ok {
		return user, fmt.Errorf("Missing claim '%s'", oidc.Name)
	}
	if mail, ok = claims[oidc.Email].(string); !ok {
		return user, fmt.Errorf("Missing claim '%s'", oidc.Email)
	}
	if username, ok = claims[oidc.PreferredUsername].(string); !ok {
		c.logger.Warn().Str("claim", oidc.PreferredUsername).Msg("Missing claim for username, falling back to email address")
		username = mail
	}
	user.DisplayName = &dn
	user.OnPremisesSamAccountName = &username
	user.Mail = &mail
	return user, nil
}

func (c cs3backend) cs3UserFromLibregraph(ctx context.Context, lu *libregraph.User) cs3.User {
	cs3id := cs3.UserId{
		Type: cs3.UserType_USER_TYPE_PRIMARY,
		Idp:  c.oidcISS,
	}

	cs3id.OpaqueId = lu.GetId()

	cs3user := cs3.User{
		Id: &cs3id,
	}
	cs3user.Username = lu.GetOnPremisesSamAccountName()
	cs3user.DisplayName = lu.GetDisplayName()
	cs3user.Mail = lu.GetMail()
	return cs3user
}
