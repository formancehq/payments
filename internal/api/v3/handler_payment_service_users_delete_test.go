package v3

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"github.com/golang/mock/gomock"
)

var _ = Describe("API v3 Payment Service Users Delete", func() {
	var (
		handlerFn http.HandlerFunc
		psuID     uuid.UUID
	)
	BeforeEach(func() {
		psuID = uuid.New()
	})

	Context("delete psu", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentServiceUsersDelete(m)
		})

		It("should return an invalid ID error when psu ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodDelete, "paymentServiceUserID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodDelete, "paymentServiceUserID", psuID.String())
			expectedErr := errors.New("psu delete error")
			m.EXPECT().PaymentServiceUsersDelete(gomock.Any(), psuID).Return(
				models.Task{}, expectedErr,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return accepted status with task ID", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodDelete, "paymentServiceUserID", psuID.String())
			taskID := models.TaskID{
				Reference:   "test",
				ConnectorID: models.ConnectorID{Reference: uuid.New(), Provider: "test"},
			}
			task := models.Task{ID: taskID}
			m.EXPECT().PaymentServiceUsersDelete(gomock.Any(), psuID).Return(task, nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusAccepted, taskID.String())
		})
	})
})
