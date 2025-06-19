package v3

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

var _ = Describe("API v3 Payments Create", func() {
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
			adj []CreatePaymentsAdjustmentsRequest
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentsCreate(m, validation.NewValidator())

			asset := "JPY/0"
			adj = []CreatePaymentsAdjustmentsRequest{
				{
					Reference: "ref_adjustment",
					CreatedAt: time.Now(),
					Amount:    big.NewInt(55),
					Asset:     &asset,
					Status:    models.PAYMENT_STATUS_PENDING.String(),
				},
			}
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
				Type:        models.PAYMENT_TYPE_PAYIN.String(),
				Scheme:      models.PAYMENT_SCHEME_CARD_AMEX.String(),
				Adjustments: adj,
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
			Entry("createdAt missing", CreatePaymentRequest{Reference: "ref", ConnectorID: testConnectorID().String()}),
			Entry("payment type missing", CreatePaymentRequest{Reference: "ref", ConnectorID: testConnectorID().String(), CreatedAt: time.Now(), Asset: "CLP/2", Amount: big.NewInt(4467)}),
			Entry("payment type invalid", CreatePaymentRequest{Reference: "ref", ConnectorID: testConnectorID().String(), CreatedAt: time.Now(), Asset: "BHD/3", Amount: big.NewInt(4467), Type: "invalid"}),
			Entry("amount missing", CreatePaymentRequest{Reference: "ref", ConnectorID: testConnectorID().String(), Type: "TRANSFER", Asset: "DFJ/0", CreatedAt: time.Now()}),
			Entry("scheme missing", CreatePaymentRequest{Reference: "ref", ConnectorID: testConnectorID().String(), CreatedAt: time.Now(), Asset: "DKK/2", Amount: big.NewInt(4467), Type: "PAYOUT"}),
			Entry("scheme invalid", CreatePaymentRequest{Reference: "ref", ConnectorID: testConnectorID().String(), CreatedAt: time.Now(), Asset: "EUR/2", Amount: big.NewInt(4467), Type: "PAYOUT", Scheme: "invalid"}),
			Entry("asset missing", CreatePaymentRequest{Reference: "ref", ConnectorID: testConnectorID().String(), CreatedAt: time.Now(), Amount: big.NewInt(4467), Type: "PAYOUT", Scheme: "CARD_VISA"}),
			Entry("asset invalid", CreatePaymentRequest{Reference: "ref", ConnectorID: testConnectorID().String(), CreatedAt: time.Now(), Asset: "wut", Amount: big.NewInt(4467), Type: "PAYOUT", Scheme: "CARD_VISA"}),
			Entry("adjustments missing", CreatePaymentRequest{Reference: "ref", ConnectorID: testConnectorID().String(), CreatedAt: time.Now(), Amount: big.NewInt(4467), Type: "PAYOUT", Scheme: "CARD_VISA", Asset: "CAD/2"}),
			Entry("adjustments missing reference", CreatePaymentRequest{Reference: "ref", ConnectorID: testConnectorID().String(), CreatedAt: time.Now(), Amount: big.NewInt(4467), Type: "PAYOUT", Scheme: "CARD_VISA", Asset: "CAD/2", Adjustments: []CreatePaymentsAdjustmentsRequest{
				{},
			}}),
			Entry("adjustments missing created at", CreatePaymentRequest{Reference: "ref", ConnectorID: testConnectorID().String(), CreatedAt: time.Now(), Amount: big.NewInt(4467), Type: "PAYOUT", Scheme: "CARD_VISA", Asset: "CAD/2", Adjustments: []CreatePaymentsAdjustmentsRequest{
				{Reference: "adj1"},
			}}),
			Entry("adjustments missing status", CreatePaymentRequest{Reference: "ref", ConnectorID: testConnectorID().String(), CreatedAt: time.Now(), Amount: big.NewInt(4467), Type: "PAYOUT", Scheme: "CARD_VISA", Asset: "CAD/2", Adjustments: []CreatePaymentsAdjustmentsRequest{
				{Reference: "adj1", CreatedAt: time.Now()},
			}}),
			Entry("adjustments missing amount", CreatePaymentRequest{Reference: "ref", ConnectorID: testConnectorID().String(), CreatedAt: time.Now(), Amount: big.NewInt(4467), Type: "PAYOUT", Scheme: "CARD_VISA", Asset: "CAD/2", Adjustments: []CreatePaymentsAdjustmentsRequest{
				{Reference: "adj1", CreatedAt: time.Now(), Status: "REFUNDED"},
			}}),
			Entry("adjustments missing asset", CreatePaymentRequest{Reference: "ref", ConnectorID: testConnectorID().String(), CreatedAt: time.Now(), Amount: big.NewInt(4467), Type: "PAYOUT", Scheme: "CARD_VISA", Asset: "CAD/2", Adjustments: []CreatePaymentsAdjustmentsRequest{
				{Reference: "adj1", CreatedAt: time.Now(), Status: "REFUNDED", Amount: big.NewInt(3)},
			}}),
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
				Type:        models.PAYMENT_TYPE_PAYIN.String(),
				Scheme:      models.PAYMENT_SCHEME_CARD_AMEX.String(),
				Adjustments: adj,
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &cpr))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status created on success", func(ctx SpecContext) {
			m.EXPECT().PaymentsCreate(gomock.Any(), gomock.Any()).Return(nil)
			cpr = CreatePaymentRequest{
				Reference:   "reference-ok",
				ConnectorID: connID.String(),
				CreatedAt:   time.Now(),
				Amount:      big.NewInt(3500),
				Asset:       "JPY/0",
				Type:        models.PAYMENT_TYPE_PAYIN.String(),
				Scheme:      models.PAYMENT_SCHEME_CARD_AMEX.String(),
				Adjustments: adj,
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &cpr))
			assertExpectedResponse(w.Result(), http.StatusCreated, "data")
		})
	})
})
