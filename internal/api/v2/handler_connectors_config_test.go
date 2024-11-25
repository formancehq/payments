package v2

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

var _ = Describe("API v2 Connectors Config", func() {
	var (
		handlerFn http.HandlerFunc
		connID    models.ConnectorID
	)
	BeforeEach(func() {
		connID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
	})

	Context("get connectors config", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = connectorsConfig(m)
		})

		It("should return an invalid ID error when connector ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "connectorID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "connectorID", connID.String())
			m.EXPECT().ConnectorsConfig(gomock.Any(), connID).Return(
				json.RawMessage("{}"), fmt.Errorf("connector configs get error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return data object", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "connectorID", connID.String())
			m.EXPECT().ConnectorsConfig(gomock.Any(), connID).Return(
				json.RawMessage("{}"), nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
