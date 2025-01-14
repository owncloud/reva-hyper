package service

import (
	"context"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/cs3org/reva/v2/pkg/events"
	"github.com/owncloud/ocis/v2/ocis-pkg/log"
	settingsmsg "github.com/owncloud/ocis/v2/protogen/gen/ocis/messages/settings/v0"
	settings "github.com/owncloud/ocis/v2/protogen/gen/ocis/services/settings/v0"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go-micro.dev/v4/client"
	"testing"
)

var testLogger = log.NewLogger()

func TestUserlogFilter_execute(t *testing.T) {
	type args struct {
		ctx       context.Context
		event     events.Event
		executant *user.UserId
		users     []string
	}
	tests := []struct {
		name string
		vc   settings.ValueService
		args args
		want []string
	}{
		{"executant", settings.MockValueService{}, args{executant: &user.UserId{OpaqueId: "executant"}, users: []string{"foo", "executant"}}, []string{"foo"}},
		{"no connection to ValueService", settings.MockValueService{
			GetValueByUniqueIdentifiersFunc: func(ctx context.Context, req *settings.GetValueByUniqueIdentifiersRequest, opts ...client.CallOption) (*settings.GetValueResponse, error) {
				return nil, errors.New("no connection to ValueService")
			},
		}, args{users: []string{"foo"}, event: events.Event{Event: events.ShareCreated{}}, ctx: context.TODO()}, []string(nil)},
		{"no setting in ValueService response", settings.MockValueService{
			GetValueByUniqueIdentifiersFunc: func(ctx context.Context, req *settings.GetValueByUniqueIdentifiersRequest, opts ...client.CallOption) (*settings.GetValueResponse, error) {
				return &settings.GetValueResponse{}, nil
			},
		}, args{users: []string{"foo"}, event: events.Event{Event: events.ShareCreated{}}, ctx: context.TODO()}, []string(nil)},
		{"ValueService nil response", settings.MockValueService{
			GetValueByUniqueIdentifiersFunc: func(ctx context.Context, req *settings.GetValueByUniqueIdentifiersRequest, opts ...client.CallOption) (*settings.GetValueResponse, error) {
				return nil, nil
			},
		}, args{users: []string{"foo"}, event: events.Event{Event: events.ShareCreated{}}, ctx: context.TODO()}, []string(nil)},
		{"event that cannot be disabled", setupMockValueService(true), args{users: []string{"foo"}, event: events.Event{Event: events.BytesReceived{}}, ctx: context.TODO()}, []string{"foo"}},
		{"ShareCreated enabled", setupMockValueService(true), args{users: []string{"foo"}, event: events.Event{Event: events.ShareCreated{}}, ctx: context.TODO()}, []string{"foo"}},
		{"ShareRemoved enabled", setupMockValueService(true), args{users: []string{"foo"}, event: events.Event{Event: events.ShareRemoved{}}, ctx: context.TODO()}, []string{"foo"}},
		{"ShareExpired enabled", setupMockValueService(true), args{users: []string{"foo"}, event: events.Event{Event: events.ShareExpired{}}, ctx: context.TODO()}, []string{"foo"}},
		{"SpaceShared enabled", setupMockValueService(true), args{users: []string{"foo"}, event: events.Event{Event: events.SpaceShared{}}, ctx: context.TODO()}, []string{"foo"}},
		{"SpaceUnshared enabled", setupMockValueService(true), args{users: []string{"foo"}, event: events.Event{Event: events.SpaceUnshared{}}, ctx: context.TODO()}, []string{"foo"}},
		{"SpaceMembershipExpired enabled", setupMockValueService(true), args{users: []string{"foo"}, event: events.Event{Event: events.SpaceMembershipExpired{}}, ctx: context.TODO()}, []string{"foo"}},
		{"SpaceDisabled enabled", setupMockValueService(true), args{users: []string{"foo"}, event: events.Event{Event: events.SpaceDisabled{}}, ctx: context.TODO()}, []string{"foo"}},
		{"SpaceDeleted enabled", setupMockValueService(true), args{users: []string{"foo"}, event: events.Event{Event: events.SpaceDeleted{}}, ctx: context.TODO()}, []string{"foo"}},
		{"PostprocessingStepFinished enabled", setupMockValueService(true), args{users: []string{"foo"}, event: events.Event{Event: events.PostprocessingStepFinished{}}, ctx: context.TODO()}, []string{"foo"}},
		{"ShareCreated disabled", setupMockValueService(false), args{users: []string{"foo"}, event: events.Event{Event: events.ShareCreated{}}, ctx: context.TODO()}, []string(nil)},
		{"ShareRemoved disabled", setupMockValueService(false), args{users: []string{"foo"}, event: events.Event{Event: events.ShareRemoved{}}, ctx: context.TODO()}, []string(nil)},
		{"ShareExpired disabled", setupMockValueService(false), args{users: []string{"foo"}, event: events.Event{Event: events.ShareExpired{}}, ctx: context.TODO()}, []string(nil)},
		{"SpaceShared disabled", setupMockValueService(false), args{users: []string{"foo"}, event: events.Event{Event: events.SpaceShared{}}, ctx: context.TODO()}, []string(nil)},
		{"SpaceUnshared disabled", setupMockValueService(false), args{users: []string{"foo"}, event: events.Event{Event: events.SpaceUnshared{}}, ctx: context.TODO()}, []string(nil)},
		{"SpaceMembershipExpired disabled", setupMockValueService(false), args{users: []string{"foo"}, event: events.Event{Event: events.SpaceMembershipExpired{}}, ctx: context.TODO()}, []string(nil)},
		{"SpaceDisabled disabled", setupMockValueService(false), args{users: []string{"foo"}, event: events.Event{Event: events.SpaceDisabled{}}, ctx: context.TODO()}, []string(nil)},
		{"SpaceDeleted disabled", setupMockValueService(false), args{users: []string{"foo"}, event: events.Event{Event: events.SpaceDeleted{}}, ctx: context.TODO()}, []string(nil)},
		{"PostprocessingStepFinished disabled", setupMockValueService(false), args{users: []string{"foo"}, event: events.Event{Event: events.PostprocessingStepFinished{}}, ctx: context.TODO()}, []string(nil)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ulf := userlogFilter{
				log:         testLogger,
				valueClient: tt.vc,
			}
			assert.Equal(t, tt.want, ulf.execute(tt.args.ctx, tt.args.event, tt.args.executant, tt.args.users))
		})
	}
}

func setupMockValueService(inApp bool) settings.ValueService {
	return settings.MockValueService{
		GetValueByUniqueIdentifiersFunc: func(ctx context.Context, req *settings.GetValueByUniqueIdentifiersRequest, opts ...client.CallOption) (*settings.GetValueResponse, error) {
			return &settings.GetValueResponse{
				Value: &settingsmsg.ValueWithIdentifier{
					Value: &settingsmsg.Value{
						Value: &settingsmsg.Value_CollectionValue{
							CollectionValue: &settingsmsg.CollectionValue{
								Values: []*settingsmsg.CollectionOption{
									{
										Key:    "in-app",
										Option: &settingsmsg.CollectionOption_BoolValue{BoolValue: inApp},
									},
								},
							},
						},
					},
				},
			}, nil
		},
	}
}
