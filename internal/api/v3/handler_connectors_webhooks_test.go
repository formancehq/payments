package v3

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Connector Webhooks", func() {
	var (
		handlerFn http.HandlerFunc
		connID    models.ConnectorID
		config    json.RawMessage
	)
	BeforeEach(func() {
		connID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		config = json.RawMessage("{}")
	})

	Context("webhooks connector", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = connectorsWebhooks(m)
		})

		It("should return a bad request error when connector ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "connectorID", "invalid")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			m.EXPECT().ConnectorsHandleWebhooks(gomock.Any(), "/", "/", gomock.Any()).Return(fmt.Errorf("connector webhooks err"))
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "connectorID", connID.String(), &config))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status ok on success", func(ctx SpecContext) {
			m.EXPECT().ConnectorsHandleWebhooks(gomock.Any(), "/", "/", gomock.Any()).Return(nil)
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "connectorID", connID.String(), &config))
			assertExpectedResponse(w.Result(), http.StatusOK, "")
		})
	})
})
