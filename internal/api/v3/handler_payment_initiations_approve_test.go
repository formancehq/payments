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

var _ = Describe("API v3 Payment Initiation Approval", func() {
	var (
		handlerFn http.HandlerFunc
		paymentID models.PaymentInitiationID
	)
	BeforeEach(func() {
		connID := models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		paymentID = models.PaymentInitiationID{Reference: "ref", ConnectorID: connID}
	})

	Context("approve payment initiation", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentInitiationsApprove(m)
		})

		It("should return a bad request error when paymentInitiationID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "paymentInitiationID", "invalid")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("payment initiation approve err")
			m.EXPECT().PaymentInitiationsApprove(gomock.Any(), gomock.Any(), false).Return(
				models.Task{},
				expectedErr,
			)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "paymentInitiationID", paymentID.String()))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status accepted on success", func(ctx SpecContext) {
			m.EXPECT().PaymentInitiationsApprove(gomock.Any(), paymentID, false).Return(
				models.Task{},
				nil,
			)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "paymentInitiationID", paymentID.String()))
			assertExpectedResponse(w.Result(), http.StatusAccepted, "data")
		})
	})
})
