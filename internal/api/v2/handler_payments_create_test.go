package v2

import (
	"errors"
	"github.com/formancehq/payments/internal/connectors/engine"
	"math/big"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v2 Payments Create", func() {
	var (
		handlerFn http.HandlerFunc
		connID    models.ConnectorID
	)
	BeforeEach(func() {
		connID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
	})

	Context("create payments", func() {
		var (
			w   *httptest.ResponseRecorder
			m   *backend.MockBackend
			cpr CreatePaymentRequest
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentsCreate(m, validation.NewValidator())
		})

		It("should return a bad request error when body is missing", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrMissingOrInvalidBody)
		})

		It("should return a bad request error when connector is not able to create payments", func(ctx SpecContext) {
			notSupportedConnectorID := models.ConnectorID{Reference: uuid.New(), Provider: "stripe"}

			expectedErr := &engine.ErrConnectorCapabilityNotSupported{Capability: "CreateFormancePayment", Provider: notSupportedConnectorID.Provider}
			m.EXPECT().PaymentsCreate(gomock.Any(), gomock.Any()).Return(expectedErr)

			cpr = CreatePaymentRequest{
				Reference:   "reference-err",
				ConnectorID: notSupportedConnectorID.String(),
				CreatedAt:   time.Now(),
				Amount:      big.NewInt(3500),
				Asset:       "JPY/0",
				Status:      models.PAYMENT_STATUS_AMOUNT_ADJUSTEMENT.String(),
				Type:        models.PAYMENT_TYPE_PAYIN.String(),
				Scheme:      models.PAYMENT_SCHEME_CARD_AMEX.String(),
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &cpr))
			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrConnectorCapabilityNotSupported)
		})

		DescribeTable("validation errors",
			func(cpr CreatePaymentRequest) {
				handlerFn(w, prepareJSONRequest(http.MethodPost, &cpr))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
			},
			Entry("reference missing", CreatePaymentRequest{}),
			Entry("connectorID missing", CreatePaymentRequest{Reference: "ref"}),
			Entry("createdAt missing", CreatePaymentRequest{Reference: "ref", ConnectorID: "id"}),
			Entry("amount missing", CreatePaymentRequest{Reference: "ref", ConnectorID: "id", CreatedAt: time.Now()}),
			Entry("payment type missing", CreatePaymentRequest{Reference: "ref", ConnectorID: "id", CreatedAt: time.Now(), Amount: big.NewInt(4467)}),
			Entry("payment type invalid", CreatePaymentRequest{Reference: "ref", ConnectorID: "id", CreatedAt: time.Now(), Amount: big.NewInt(4467), Type: "invalid"}),
			Entry("scheme missing", CreatePaymentRequest{Reference: "ref", ConnectorID: "id", CreatedAt: time.Now(), Amount: big.NewInt(4467), Type: "PAYOUT"}),
			Entry("scheme invalid", CreatePaymentRequest{Reference: "ref", ConnectorID: "id", CreatedAt: time.Now(), Amount: big.NewInt(4467), Type: "PAYOUT", Scheme: "invalid"}),
			Entry("asset missing", CreatePaymentRequest{Reference: "ref", ConnectorID: "id", CreatedAt: time.Now(), Amount: big.NewInt(4467), Type: "PAYOUT", Scheme: "CARD_VISA"}),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("payment create err")
			m.EXPECT().PaymentsCreate(gomock.Any(), gomock.Any()).Return(expectedErr)
			cpr = CreatePaymentRequest{
				Reference:   "reference-err",
				ConnectorID: connID.String(),
				CreatedAt:   time.Now(),
				Amount:      big.NewInt(3500),
				Asset:       "JPY/0",
				Status:      models.PAYMENT_STATUS_AMOUNT_ADJUSTEMENT.String(),
				Type:        models.PAYMENT_TYPE_PAYIN.String(),
				Scheme:      models.PAYMENT_SCHEME_CARD_AMEX.String(),
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &cpr))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status OK on success", func(ctx SpecContext) {
			m.EXPECT().PaymentsCreate(gomock.Any(), gomock.Any()).Return(nil)
			cpr = CreatePaymentRequest{
				Reference:   "reference-ok",
				ConnectorID: connID.String(),
				CreatedAt:   time.Now(),
				Amount:      big.NewInt(3500),
				Asset:       "JPY/0",
				Status:      models.PAYMENT_STATUS_AMOUNT_ADJUSTEMENT.String(),
				Type:        models.PAYMENT_TYPE_PAYIN.String(),
				Scheme:      models.PAYMENT_SCHEME_CARD_AMEX.String(),
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &cpr))
			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
