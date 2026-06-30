package v3

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/storage"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Connectors uninstall", func() {
	var (
		handlerFn http.HandlerFunc
		connID    models.ConnectorID
	)
	BeforeEach(func() {
		connID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
	})

	Context("uninstall connectors", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = connectorsUninstall(m)
		})

		It("should return a bad request error when connector ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "connectorID", "invalid")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("connectors uninstall err")
			m.EXPECT().ConnectorsUninstall(gomock.Any(), gomock.Any()).Return(models.Task{}, expectedErr)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "connectorID", connID.String()))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		// A non-existent connector trips the tasks->connectors foreign key when the
		// uninstall task is upserted; this must be a 4xx, not a 500 (EN-1344 / CU-S3).
		It("should return a 4xx (not a 500) when the connector does not exist", func(ctx SpecContext) {
			m.EXPECT().ConnectorsUninstall(gomock.Any(), gomock.Any()).Return(models.Task{}, storage.ErrForeignKeyViolation)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "connectorID", connID.String()))
			assertExpectedResponse(w.Result(), http.StatusBadRequest, "VALIDATION")
		})

		It("should map a not found error to 404", func(ctx SpecContext) {
			m.EXPECT().ConnectorsUninstall(gomock.Any(), gomock.Any()).Return(models.Task{}, storage.ErrNotFound)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "connectorID", connID.String()))
			assertExpectedResponse(w.Result(), http.StatusNotFound, "NOT_FOUND")
		})

		It("should return status accepted on success", func(ctx SpecContext) {
			m.EXPECT().ConnectorsUninstall(gomock.Any(), connID).Return(models.Task{}, nil)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "connectorID", connID.String()))
			assertExpectedResponse(w.Result(), http.StatusAccepted, "data")
		})
	})
})
