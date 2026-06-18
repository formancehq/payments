package v2

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/formancehq/go-libs/v5/pkg/authn/jwt"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v2 Router", func() {
	// Routing-level tests: requests go through the chi router (newRouter) rather
	// than calling handlers directly, so the URL path is parsed into chi
	// URLParams the same way it is in production. This guards against route
	// declaration bugs that handler-only tests cannot catch (EN-1091: the route
	// declared a literal "connectorID" segment instead of a "{connectorID}"
	// param, so chi.URLParam(r, "connectorID") was always empty).
	Context("public connector webhooks route", func() {
		var (
			r      http.Handler
			m      *backend.MockBackend
			connID models.ConnectorID
		)
		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			// Build the router with a real authenticator that DENIES every
			// request. The webhooks route is public, so requests must reach the
			// handler regardless of authentication. If the route were ever moved
			// behind the JWT middleware, this authenticator would reject the
			// request with 401 and these tests would fail.
			auth := jwt.NewMockAuthenticator(ctrl)
			auth.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(false, nil).AnyTimes()
			r = newRouter(m, auth, false)
			connID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		})

		It("routes the connector ID from the path to the handler", func(ctx SpecContext) {
			m.EXPECT().
				ConnectorsHandleWebhooks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ any, _ string, _ string, webhook models.Webhook) error {
					Expect(webhook.ConnectorID.String()).To(Equal(connID.String()))
					return nil
				})

			req := httptest.NewRequest(http.MethodPost, "/connectors/webhooks/"+connID.String()+"/", strings.NewReader("{}"))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			Expect(w.Result().StatusCode).To(Equal(http.StatusOK))
		})

		It("routes webhook sub-paths via the wildcard", func(ctx SpecContext) {
			m.EXPECT().
				ConnectorsHandleWebhooks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil)

			req := httptest.NewRequest(http.MethodPost, "/connectors/webhooks/"+connID.String()+"/some/provider/path", strings.NewReader("{}"))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			Expect(w.Result().StatusCode).To(Equal(http.StatusOK))
		})

		It("returns a bad request when the connector ID in the path is invalid", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodPost, "/connectors/webhooks/invalid/", strings.NewReader("{}"))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			Expect(w.Result().StatusCode).To(Equal(http.StatusBadRequest))
		})
	})
})
