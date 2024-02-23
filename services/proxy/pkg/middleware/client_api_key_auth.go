package middleware

import (
	"encoding/json"
	"errors"
	revactx "github.com/cs3org/reva/v2/pkg/ctx"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/owncloud/ocis/v2/ocis-pkg/log"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/user/backend"
	"go-micro.dev/v4/store"
	"net/http"
)

// ClientAPIKeyAuthenticator is the authenticator responsible for client API keys.
type ClientAPIKeyAuthenticator struct {
	Logger       log.Logger
	UserProvider backend.UserBackend
	SigningKey   string
	Store        store.Store
}

type ClientAPIKeyRecord struct {
	UserId string
	Scopes []string
}

func (m ClientAPIKeyAuthenticator) CreateClientAPIKey() (string, string, error) {
	k := uuid.New()
	signature, err := jwt.SigningMethodHS256.Sign(k.String(), []byte(m.SigningKey))
	if err != nil {
		return "", "", err
	}
	return k.String(), signature, nil
}

func (m ClientAPIKeyAuthenticator) SaveKey(userId string, keyid string) error {
	record := ClientAPIKeyRecord{
		UserId: userId,
	}
	bytes, err := json.Marshal(record)
	if err != nil {
		return err
	}
	r := store.Record{
		Key:   keyid,
		Value: bytes,
	}
	return m.Store.Write(&r)
}

// Authenticate implements the authenticator interface to authenticate requests via basic auth.
func (m ClientAPIKeyAuthenticator) Authenticate(r *http.Request) (*http.Request, bool) {
	if isPublicPath(r.URL.Path) {
		// The authentication of public path requests is handled by another authenticator.
		// Since we can't guarantee the order of execution of the authenticators, we better
		// implement an early return here for paths we can't authenticate in this authenticator.
		return nil, false
	}

	clientApiKeyId, clientApiKeySecret, ok := r.BasicAuth()
	if !ok {
		return nil, false
	}

	// re-build JWT
	err := m.VerifyBasicAuth(clientApiKeyId, clientApiKeySecret)
	if err != nil {
		m.Logger.Error().
			Err(err).
			Str("authenticator", "client_api_keys").
			Str("path", r.URL.Path).
			Msg("failed to authenticate request")
		return nil, false
	}

	// lookup user data in micro store
	clientAPIKeyRecord, err := m.LookupUser(clientApiKeyId)
	if err != nil {
		m.Logger.Error().
			Err(err).
			Str("authenticator", "client_api_keys").
			Str("path", r.URL.Path).
			Msg("failed to lookup client api key record")
		return nil, false
	}

	_, token, err := m.UserProvider.GetUserByClaims(r.Context(), "userid", clientAPIKeyRecord.UserId)
	if err != nil {
		m.Logger.Error().
			Err(err).
			Str("authenticator", "client_api_keys").
			Str("path", r.URL.Path).
			Str("userid", clientAPIKeyRecord.UserId).
			Msg("failed to get user by userid")
		return nil, false
	}

	// set token in request
	r.Header.Set(revactx.TokenHeader, token)

	m.Logger.Debug().
		Str("authenticator", "client_api_keys").
		Str("path", r.URL.Path).
		Msg("successfully authenticated request")
	return r, true
}

func (m ClientAPIKeyAuthenticator) VerifyBasicAuth(username string, password string) error {
	signature, err := jwt.SigningMethodHS256.Sign(username, []byte(m.SigningKey))
	if err != nil {
		return err
	}
	if signature != password {
		return errors.New("client api key secret not matching the signature")
	}
	return nil
}

func (m ClientAPIKeyAuthenticator) LookupUser(clientApiKeyId string) (*ClientAPIKeyRecord, error) {
	records, err := m.Store.Read(clientApiKeyId)
	if err != nil {
		return nil, err
	}
	if len(records) < 1 {
		return nil, store.ErrNotFound
	}
	data := records[0].Value
	v := ClientAPIKeyRecord{}
	err = json.Unmarshal(data, &v)
	if err != nil {
		return nil, err
	}

	return &v, nil
}
