package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"sync"
	"sync/atomic"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	group "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/cs3org/reva/v2/pkg/events"
	"github.com/cs3org/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/owncloud/ocis/v2/ocis-pkg/l10n"
	"github.com/owncloud/ocis/v2/ocis-pkg/log"
	"github.com/owncloud/ocis/v2/ocis-pkg/middleware"
	settingssvc "github.com/owncloud/ocis/v2/protogen/gen/ocis/services/settings/v0"
	"github.com/owncloud/ocis/v2/services/notifications/pkg/channels"
	"github.com/owncloud/ocis/v2/services/notifications/pkg/email"
	"github.com/owncloud/ocis/v2/services/settings/pkg/store/defaults"
	"go-micro.dev/v4/metadata"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// Service should be named `Runner`
type Service interface {
	Run() error
	Close()
}

// NewEventsNotifier provides a new eventsNotifier
func NewEventsNotifier(
	events <-chan events.Event,
	channel channels.Channel,
	logger log.Logger,
	gatewaySelector pool.Selectable[gateway.GatewayAPIClient],
	valueService settingssvc.ValueService,
	serviceAccountID, serviceAccountSecret, emailTemplatePath, defaultLanguage, ocisURL string) Service {

	return eventsNotifier{
		logger:               logger,
		channel:              channel,
		events:               events,
		gatewaySelector:      gatewaySelector,
		valueService:         valueService,
		serviceAccountID:     serviceAccountID,
		serviceAccountSecret: serviceAccountSecret,
		emailTemplatePath:    emailTemplatePath,
		defaultLanguage:      defaultLanguage,
		ocisURL:              ocisURL,
		stopCh:               make(chan struct{}, 1),
		stopped:              new(atomic.Bool),
	}
}

type eventsNotifier struct {
	logger               log.Logger
	channel              channels.Channel
	events               <-chan events.Event
	gatewaySelector      pool.Selectable[gateway.GatewayAPIClient]
	valueService         settingssvc.ValueService
	emailTemplatePath    string
	translationPath      string
	defaultLanguage      string
	ocisURL              string
	serviceAccountID     string
	serviceAccountSecret string
	stopCh               chan struct{}
	stopped              *atomic.Bool
}

func (s eventsNotifier) Run() error {
	var wg sync.WaitGroup

	s.logger.Debug().
		Msg("eventsNotifier started")
EventLoop:
	for {
		select {
		case evt, ok := <-s.events:
			if !ok {
				break EventLoop
			}
			// TODO: needs to be replaced with a worker pool
			wg.Add(1)
			go func() {
				defer wg.Done()

				switch e := evt.Event.(type) {
				case events.SpaceShared:
					s.handleSpaceShared(e)
				case events.SpaceUnshared:
					s.handleSpaceUnshared(e)
				case events.SpaceMembershipExpired:
					s.handleSpaceMembershipExpired(e)
				case events.ShareCreated:
					s.handleShareCreated(e)
				case events.ShareExpired:
					s.handleShareExpired(e)
				}
			}()

			if s.stopped.Load() {
				break EventLoop
			}
		case <-s.stopCh:
			s.logger.Debug().
				Msg("eventsNotifier stopped")
			break EventLoop
		}
	}
	// wait until all the goroutines processing events have finished
	wg.Wait()
	return nil
}

func (s eventsNotifier) Close() {
	if s.stopped.CompareAndSwap(false, true) {
		close(s.stopCh)
	}
}

func (s eventsNotifier) render(ctx context.Context, template email.MessageTemplate,
	granteeFieldName string, fields map[string]string, granteeList []*user.User, sender string) ([]*channels.Message, error) {
	// Render the Email Template for each user
	messageList := make([]*channels.Message, len(granteeList))
	for i, usr := range granteeList {
		locale := l10n.MustGetUserLocale(ctx, usr.GetId().GetOpaqueId(), "", s.valueService)
		fields[granteeFieldName] = usr.GetDisplayName()

		rendered, err := email.RenderEmailTemplate(template, locale, s.defaultLanguage, s.emailTemplatePath, s.translationPath, fields)
		if err != nil {
			return nil, err
		}
		rendered.Sender = sender
		rendered.Recipient = []string{usr.GetMail()}
		messageList[i] = rendered
	}
	return messageList, nil
}

func (s eventsNotifier) send(ctx context.Context, recipientList []*channels.Message) {
	for _, r := range recipientList {
		err := s.channel.SendMessage(ctx, r)
		if err != nil {
			s.logger.Error().Err(err).Str("event", "SendEmail").Msg("failed to send a message")
		}
	}
}

func (s eventsNotifier) ensureGranteeList(ctx context.Context, executant, u *user.UserId, g *group.GroupId) []*user.User {
	granteeList, err := s.getGranteeList(ctx, executant, u, g)
	if err != nil {
		s.logger.Error().Err(err).Str("event", "ensureGranteeList").Msg("Could not get grantee list")
		return nil
	} else if len(granteeList) == 0 {
		return nil
	}
	return granteeList
}

func (s eventsNotifier) getGranteeList(ctx context.Context, executant, u *user.UserId, g *group.GroupId) ([]*user.User, error) {
	switch {
	case u != nil:
		if s.disableEmails(ctx, u) {
			return nil, nil
		}
		usr, err := s.getUser(ctx, u)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(usr.GetMail()) == "" {
			s.logger.Debug().Str("event", "getGranteeList").Msgf("User %s has no email, skipped", usr.GetUsername())
			return nil, nil
		}
		return []*user.User{usr}, nil
	case g != nil:
		gatewayClient, err := s.gatewaySelector.Next()
		if err != nil {
			return nil, err
		}

		res, err := gatewayClient.GetGroup(ctx, &group.GetGroupRequest{GroupId: g})
		if err != nil {
			return nil, err
		}
		if res.GetStatus().GetCode() != rpc.Code_CODE_OK {
			return nil, errors.New("could not get group")
		}

		userList := make([]*user.User, 0, len(res.GetGroup().GetMembers()))
		for _, userID := range res.GetGroup().GetMembers() {
			// don't add the executant
			if userID.GetOpaqueId() == executant.GetOpaqueId() {
				continue
			}
			// don't add users who opted out
			if s.disableEmails(ctx, userID) {
				continue
			}
			usr, err := s.getUser(ctx, userID)
			if err != nil {
				return nil, err
			}
			if strings.TrimSpace(usr.GetMail()) == "" {
				s.logger.Debug().Str("event", "getGranteeList").Msgf("User %s has no email, skipped", usr.GetUsername())
				continue
			}
			userList = append(userList, usr)
		}
		return userList, nil
	default:
		return nil, errors.New("need at least one non-nil grantee")
	}
}

func (s eventsNotifier) getUser(ctx context.Context, u *user.UserId) (*user.User, error) {
	if u == nil {
		return nil, errors.New("need at least one non-nil grantee")
	}
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	r, err := gatewayClient.GetUser(ctx, &user.GetUserRequest{UserId: u})
	if err != nil {
		return nil, err
	}
	if r.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return nil, fmt.Errorf("unexpected status code from gateway client: %d", r.GetStatus().GetCode())
	}
	return r.GetUser(), nil
}

func (s eventsNotifier) disableEmails(ctx context.Context, u *user.UserId) bool {
	granteeCtx := metadata.Set(ctx, middleware.AccountID, u.OpaqueId)
	if resp, err := s.valueService.GetValueByUniqueIdentifiers(granteeCtx,
		&settingssvc.GetValueByUniqueIdentifiersRequest{
			AccountUuid: u.OpaqueId,
			SettingId:   defaults.SettingUUIDProfileDisableNotifications,
		},
	); err == nil {
		return resp.GetValue().GetValue().GetBoolValue()

	}
	return false
}

func (s eventsNotifier) getResourceInfo(ctx context.Context, resourceID *provider.ResourceId, fieldmask *fieldmaskpb.FieldMask) (*provider.ResourceInfo, error) {
	// TODO: maybe cache this stat to reduce storage iops
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	md, err := gatewayClient.Stat(ctx, &provider.StatRequest{
		Ref: &provider.Reference{
			ResourceId: resourceID,
		},
		FieldMask: fieldmask,
	})

	if err != nil {
		return nil, err
	}
	if md.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return nil, fmt.Errorf("could not resource info: %s", md.Status.Message)
	}
	return md.GetInfo(), nil
}

// TODO: this function is a backport for go1.19 url.JoinPath, upon go bump, replace this
func urlJoinPath(base string, elements ...string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(append([]string{u.Path}, elements...)...)
	return u.String(), nil
}
