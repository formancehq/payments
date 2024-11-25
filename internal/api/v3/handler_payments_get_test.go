package v3

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Payments", func() {
	var (
		handlerFn http.HandlerFunc
		payID     models.PaymentID
	)
	BeforeEach(func() {
		connID := models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		payRef := models.PaymentReference{Reference: "ref", Type: models.PAYMENT_TYPE_TRANSFER}
		payID = models.PaymentID{PaymentReference: payRef, ConnectorID: connID}
	})

	Context("get payments", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentsGet(m)
		})

		It("should return an invalid ID error when payment ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "paymentID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "paymentID", payID.String())
			m.EXPECT().PaymentsGet(gomock.Any(), payID).Return(
				&models.Payment{}, fmt.Errorf("payments get error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return data object", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "paymentID", payID.String())
			m.EXPECT().PaymentsGet(gomock.Any(), payID).Return(
				&models.Payment{}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
