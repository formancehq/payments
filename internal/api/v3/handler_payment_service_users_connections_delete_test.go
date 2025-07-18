package v3

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Payment Service Users Connections Delete", func() {
	var (
		handlerFn    http.HandlerFunc
		psuID        uuid.UUID
		connectorID  models.ConnectorID
		connectionID string
	)
	BeforeEach(func() {
		psuID = uuid.New()
		connectorID = models.ConnectorID{Reference: uuid.New(), Provider: "test"}
		connectionID = "test-connection-id"
	})

	Context("delete psu connection", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentServiceUsersDeleteConnection(m)
		})

		It("should return an invalid ID error when psu ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodDelete, "paymentServiceUserID", "invalidvalue", "connectorID", connectorID.String(), "connectionID", connectionID)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an invalid ID error when connector ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodDelete, "paymentServiceUserID", psuID.String(), "connectorID", "invalidvalue", "connectionID", connectionID)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodDelete, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String(), "connectionID", connectionID)
			expectedErr := errors.New("psu connection delete error")
			m.EXPECT().PaymentServiceUsersConnectionsDelete(gomock.Any(), connectorID, psuID, connectionID).Return(
				models.Task{}, expectedErr,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return accepted status with task ID", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodDelete, "paymentServiceUserID", psuID.String(), "connectorID", connectorID.String(), "connectionID", connectionID)
			taskID := models.TaskID{
				Reference:   "test",
				ConnectorID: connectorID,
			}
			task := models.Task{ID: taskID}
			m.EXPECT().PaymentServiceUsersConnectionsDelete(gomock.Any(), connectorID, psuID, connectionID).Return(task, nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusAccepted, taskID.String())
		})
	})
})
