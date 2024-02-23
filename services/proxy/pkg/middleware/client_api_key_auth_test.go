package middleware

import (
	revactx "github.com/cs3org/reva/v2/pkg/ctx"
	"go-micro.dev/v4/store"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/gomega"
	"github.com/owncloud/ocis/v2/ocis-pkg/log"
	"github.com/owncloud/ocis/v2/services/proxy/pkg/user/backend/mocks"
)

var _ = Describe("Authenticating requests", Label("ClientAPIKeyAuthenticator"), func() {
	var authenticator ClientAPIKeyAuthenticator
	ub := mocks.UserBackend{}
	ub.On("GetUserByClaims", mock.Anything, mock.Anything, mock.Anything).Return(
		nil,
		"reva-token",
		nil,
	)

	BeforeEach(func() {
		authenticator = ClientAPIKeyAuthenticator{
			Logger:       log.NewLogger(),
			UserProvider: &ub,
			SigningKey:   "lorwm",
			Store:        store.NewMemoryStore(),
		}
	})

	When("the request contains correct data", func() {
		It("should successfully authenticate", func() {
			// user creates client api key
			key, s, err := authenticator.CreateClientAPIKey()
			Expect(err).To(BeNil())

			err = authenticator.SaveKey("einstein", key)
			Expect(err).To(BeNil())

			// call api with client api key
			req := httptest.NewRequest(http.MethodGet, "http://example.com/example/path", http.NoBody)
			req.SetBasicAuth(key, s)

			req2, valid := authenticator.Authenticate(req)

			Expect(valid).To(Equal(true))
			Expect(req2).ToNot(BeNil())
			Expect(req2.Header.Get(revactx.TokenHeader)).To(Equal("reva-token"))
		})
	})
})
