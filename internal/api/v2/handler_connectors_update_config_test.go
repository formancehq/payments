package v2

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"github.com/golang/mock/gomock"
)

var _ = Describe("API v2 Connectors Update Config", func() {
	var (
		handlerFn http.HandlerFunc
		connID    models.ConnectorID
		config    = json.RawMessage("{}")
	)
	BeforeEach(func() {
		connID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
	})

	Context("reset connectors", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = connectorsConfigUpdate(m)
		})

		It("should return a bad request error when connector ID is invalid", func(ctx SpecContext) {
			req := prepareJSONRequestWithQuery(http.MethodGet, "connectorID", "invalid", &config)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("connectors reset err")
			m.EXPECT().ConnectorsConfigUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedErr)
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodGet, "connectorID", connID.String(), &config))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status no content on success", func(ctx SpecContext) {
			m.EXPECT().ConnectorsConfigUpdate(gomock.Any(), connID, gomock.Any()).Return(nil)
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodGet, "connectorID", connID.String(), config))
			assertExpectedResponse(w.Result(), http.StatusNoContent, "")
		})
	})
})
