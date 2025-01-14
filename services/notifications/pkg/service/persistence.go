package service

import (
	"context"
	"encoding/json"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/owncloud/ocis/v2/ocis-pkg/log"
	v0 "github.com/owncloud/ocis/v2/protogen/gen/ocis/messages/eventhistory/v0"
	ehsvc "github.com/owncloud/ocis/v2/protogen/gen/ocis/services/eventhistory/v0"
	"github.com/pkg/errors"
	"go-micro.dev/v4/store"
)

type userEventStore struct {
	log           log.Logger
	store         store.Store
	historyClient ehsvc.EventHistoryService
}

type userEventIds struct {
	User     *user.User `json:"user"`
	EventIds []string   `json:"event_ids"`
}

type userEvents struct {
	User   *user.User
	Events []*v0.Event
}

const (
	_intervalDaily  = "daily"
	_intervalWeekly = "weekly"
)

func newUserEventStore(l log.Logger, s store.Store, hc ehsvc.EventHistoryService) *userEventStore {
	return &userEventStore{log: l, store: s, historyClient: hc}
}

func (s *userEventStore) persist(interval string, eventId string, users []*user.User) []*user.User {
	var errorUsers []*user.User
	for _, u := range users {
		key := interval + "_" + u.Id.OpaqueId

		// Note: This is not thread safe and can result in missing events
		records, err := s.store.Read(key)
		if err != nil && err != store.ErrNotFound {
			s.log.Error().Err(err).Str("eventId", eventId).Str("userId", u.Id.OpaqueId).Msg("cannot read record")
			errorUsers = append(errorUsers, u)
			continue
		}
		var record userEventIds
		if len(records) == 0 {
			record = userEventIds{}
		} else {
			if err = json.Unmarshal(records[0].Value, &record); err != nil {
				s.log.Warn().Err(err).Str("eventId", eventId).Str("userId", u.Id.OpaqueId).Msg("cannot unmarshal json")
				errorUsers = append(errorUsers, u)
				continue
			}
		}
		record.User = u
		record.EventIds = append(record.EventIds, eventId)
		b, err := json.Marshal(record)
		if err != nil {
			s.log.Warn().Err(err).Str("eventId", eventId).Str("userId", u.Id.OpaqueId).Msg("cannot marshal record")
			errorUsers = append(errorUsers, u)
			continue
		}
		err = s.store.Write(&store.Record{
			Key:   key,
			Value: b,
		})
		if err != nil {
			s.log.Error().Err(err).Str("eventId", eventId).Str("userId", u.Id.OpaqueId).Msg("cannot write record")
			errorUsers = append(errorUsers, u)
			continue
		}

	}
	return errorUsers
}

func (s *userEventStore) listKeys(prefix string) ([]string, error) {
	return s.store.List(store.ListPrefix(prefix))
}

func (s *userEventStore) pop(ctx context.Context, key string) (*userEvents, error) {
	records, err := s.store.Read(key)
	if err != nil && err != store.ErrNotFound {
		return nil, errors.New("cannot get records")
	}
	if len(records) == 0 {
		return nil, errors.New("no records found")
	}
	var record userEventIds
	err = json.Unmarshal(records[0].Value, &record)
	if err != nil {
		s.log.Warn().Err(err).Str("key", key).Msg("cannot unmarshal json")
		return nil, err
	}

	res, err := s.historyClient.GetEvents(ctx, &ehsvc.GetEventsRequest{Ids: record.EventIds})
	if err != nil {
		s.log.Error().Err(err).Strs("eventIds", record.EventIds).Msg("cannot get events")
		return nil, err
	}
	err = s.store.Delete(key)
	if err != nil {
		s.log.Error().Err(err).Strs("eventIds", record.EventIds).Msg("cannot delete records")
		return nil, err
	}
	return &userEvents{
		User:   record.User,
		Events: res.GetEvents(),
	}, nil
}
