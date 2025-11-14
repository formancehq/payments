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

var _ = Describe("API v3 Payment Initiation Get", func() {
	var (
		handlerFn http.HandlerFunc
		paymentID models.PaymentInitiationID
	)
	BeforeEach(func() {
		connID := models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		paymentID = models.PaymentInitiationID{Reference: "ref", ConnectorID: connID}
	})

	Context("get payment initiation", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentInitiationsGet(m)
		})

		It("should return a bad request error when paymentInitiationID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "paymentInitiationID", "invalid")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("payment initiation get err")
			m.EXPECT().PaymentInitiationsGet(gomock.Any(), gomock.Any()).Return(
				&models.PaymentInitiation{},
				expectedErr,
			)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "paymentInitiationID", paymentID.String()))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return an internal server error when backend returns error finding payment adjustment", func(ctx SpecContext) {
			expectedErr := errors.New("payment initiation get adjustment err")
			m.EXPECT().PaymentInitiationsGet(gomock.Any(), gomock.Any()).Return(
				&models.PaymentInitiation{},
				nil,
			)
			m.EXPECT().PaymentInitiationAdjustmentsGetLast(gomock.Any(), paymentID).Return(
				&models.PaymentInitiationAdjustment{},
				expectedErr,
			)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "paymentInitiationID", paymentID.String()))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status ok on success", func(ctx SpecContext) {
			m.EXPECT().PaymentInitiationsGet(gomock.Any(), paymentID).Return(
				&models.PaymentInitiation{},
				nil,
			)
			m.EXPECT().PaymentInitiationAdjustmentsGetLast(gomock.Any(), paymentID).Return(
				&models.PaymentInitiationAdjustment{},
				nil,
			)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "paymentInitiationID", paymentID.String()))
			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
