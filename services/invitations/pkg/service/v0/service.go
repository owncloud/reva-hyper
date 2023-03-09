package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	libregraph "github.com/owncloud/libre-graph-api-go"
	"github.com/owncloud/ocis/v2/ocis-pkg/log"
	"github.com/owncloud/ocis/v2/services/invitations/pkg/config"
	"github.com/owncloud/ocis/v2/services/invitations/pkg/invitations"
)

const (
	OwnCloudInstanceRel = "http://invitations.owncloud/rel/server-instance"
	OpenIDConnectRel    = "http://openid.net/specs/connect/1.0/issuer"
)

// Service defines the extension handlers.
type Service interface {
	// Invite creates a new invitation. Invitation adds an external user to the organization.
	//
	// When creating a new invitation you have several options available:
	// 1. On invitation creation, Microsoft Graph can automatically send an
	//    invitation email directly to the invited user, or your app can use
	//    the inviteRedeemUrl returned in the creation response to craft your
	//    own invitation (through your communication mechanism of choice) to
	//    the invited user. If you decide to have Microsoft Graph send an
	//    invitation email automatically, you can control the content and
	//    language of the email using invitedUserMessageInfo.
	// 2. When the user is invited, a user entity (of userType Guest) is
	//    created and can now be used to control access to resources. The
	//    invited user has to go through the redemption process to access any
	//    resources they have been invited to.
	Invite(ctx context.Context, invitation *invitations.Invitation) (*invitations.Invitation, error)
}

// New returns a new instance of Service
func New(opts ...Option) (Service, error) {
	options := newOptions(opts...)

	urlTemplate, err := template.New("invitations-provisioning-endpoint-url").Parse(options.Config.Endpoint.URL)
	bodyTemplate, err := template.New("invitations-provisioning-endpoint-url").Parse(options.Config.Endpoint.BodyTemplate)
	if err != nil {
		return nil, err
	}
	return svc{
		log:          options.Logger,
		config:       options.Config,
		urlTemplate:  urlTemplate,
		bodyTemplate: bodyTemplate,
	}, nil
}

type svc struct {
	config       *config.Config
	log          log.Logger
	urlTemplate  *template.Template
	bodyTemplate *template.Template
}

// Invite implements the service interface
func (s svc) Invite(ctx context.Context, invitation *invitations.Invitation) (*invitations.Invitation, error) {

	if invitation == nil {
		return nil, ErrBadRequest
	}

	if invitation.InvitedUserEmailAddress == "" {
		return nil, ErrMissingEmail
	}

	user := &libregraph.User{
		Mail: &invitation.InvitedUserEmailAddress,
		// TODO we cannot set the user type here
	}

	if invitation.InvitedUserDisplayName != "" {
		user.DisplayName = &invitation.InvitedUserDisplayName
	}
	// we don't really need a username as guests have to log in with their email address anyway
	// what if later a user is provisioned with a guest accounts email address?

	templateVars := map[string]string{
		"redirectUrl": invitation.InviteRedirectUrl,
		// TODO message and other options
		"mail":        invitation.InvitedUserEmailAddress,
		"displayName": invitation.InvitedUserDisplayName,
		"userType":    invitation.InvitedUserType,
	}

	var urlWriter strings.Builder
	if err := s.urlTemplate.Execute(&urlWriter, templateVars); err != nil {
		return nil, err
	}

	var bodyWriter strings.Builder
	if err := s.bodyTemplate.Execute(&bodyWriter, templateVars); err != nil {
		return nil, err
	}

	// send a request to the provisioning endpoint
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true /*TODO make configurable*/},
	}
	client := &http.Client{Transport: tr}

	userRole := "guest"
	educationUser := libregraph.EducationUser{
		DisplayName:              &invitation.InvitedUserDisplayName,
		Mail:                     &invitation.InvitedUserEmailAddress,
		OnPremisesSamAccountName: &invitation.InvitedUserEmailAddress,
		PrimaryRole:              &userRole,
	}

	jsonBody, err := json.Marshal(educationUser)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(s.config.Endpoint.Method, s.config.Endpoint.URL, bytes.NewBufferString(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	// TODO either forward current user token or use bearer token?
	switch s.config.Endpoint.Authorization {
	case "token":
		// TODO forward current reva access token
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+s.config.Endpoint.Token)
	default:
		return nil, fmt.Errorf("unknown authorization: " + s.config.Endpoint.Authorization)
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error when sending user creation request, got %d response from remote", res.StatusCode)
	}
	defer res.Body.Close()

	// TODO hm ok so we expect the rosponse to be a libregraph user ... so much for a generic endpoint
	// we could try parsing into a map[string]interface{} .... hm ... maybe better to be specific about
	// the actual backend: libregraph, keycloak, scim or even oc10?

	// Or we remember the mail of the user in memory and try to check if the user is already avilable via
	// a local user api ... hm ... graph or cs3 user backend now?

	// in any case this will require an additional endpoint to keep track of the ongoing invitations

	invitedUser := &libregraph.User{}
	err = json.NewDecoder(res.Body).Decode(invitedUser)
	if err != nil {
		return nil, err
	}

	response := &invitations.Invitation{
		InvitedUser: invitedUser,
		Status:      "Completed",
	}

	// optionally send an email

	return response, nil
}
