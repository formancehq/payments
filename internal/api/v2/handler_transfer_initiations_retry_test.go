package v2

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

var _ = Describe("API v2 Payment Initiation Retry", func() {
	var (
		handlerFn http.HandlerFunc
		paymentID models.PaymentInitiationID
	)
	BeforeEach(func() {
		connID := models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		paymentID = models.PaymentInitiationID{Reference: "ref", ConnectorID: connID}
	})

	Context("retry payment initiation", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = transferInitiationsRetry(m)
		})

		It("should return a bad request error when transferInitiationID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest("transferInitiationID", "invalid")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("payment initiation retry err")
			m.EXPECT().PaymentInitiationsRetry(gomock.Any(), gomock.Any(), true).Return(
				models.Task{},
				expectedErr,
			)
			handlerFn(w, prepareQueryRequest("transferInitiationID", paymentID.String()))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status no content on success", func(ctx SpecContext) {
			m.EXPECT().PaymentInitiationsRetry(gomock.Any(), paymentID, true).Return(
				models.Task{},
				nil,
			)
			handlerFn(w, prepareQueryRequest("transferInitiationID", paymentID.String()))
			assertExpectedResponse(w.Result(), http.StatusNoContent, "")
		})
	})
})
