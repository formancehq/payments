package v3

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Payment Initiation Payments List", func() {
	var (
		handlerFn http.HandlerFunc
		paymentID models.PaymentInitiationID
	)
	BeforeEach(func() {
		connID := models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		paymentID = models.PaymentInitiationID{Reference: "ref", ConnectorID: connID}
	})

	Context("list payment initiation list payments", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentInitiationPaymentsList(m)
		})

		It("should return a validation request error when paymentInitiationID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "paymentInitiationID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "paymentInitiationID", paymentID.String())
			m.EXPECT().PaymentInitiationRelatedPaymentsList(gomock.Any(), paymentID, gomock.Any()).Return(
				&bunpaginate.Cursor[models.Payment]{}, fmt.Errorf("payment initiation payments list error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return a cursor object", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "paymentInitiationID", paymentID.String())
			m.EXPECT().PaymentInitiationRelatedPaymentsList(gomock.Any(), paymentID, gomock.Any()).Return(
				&bunpaginate.Cursor[models.Payment]{}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "cursor")
		})
	})
})
