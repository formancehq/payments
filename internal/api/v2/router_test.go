package v2

import (
	"net/http"
	"net/http/httptest"
	"strings"

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
			// authenticator is nil: the webhooks route is public so the JWT
			// middleware closure is never invoked for it.
			r = newRouter(m, nil, false)
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