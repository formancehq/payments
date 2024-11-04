package v3

import (
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/formancehq/payments/internal/api/backend"
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
			cpr createPaymentRequest
			adj []createPaymentsAdjustmentsRequest
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentsCreate(m)

			asset := "JPY"
			adj = []createPaymentsAdjustmentsRequest{
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

		DescribeTable("validation errors",
			func(cpr createPaymentRequest) {
				handlerFn(w, prepareJSONRequest(http.MethodPost, &cpr))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
			},
			Entry("reference missing", createPaymentRequest{}),
			Entry("connectorID missing", createPaymentRequest{Reference: "ref"}),
			Entry("createdAt missing", createPaymentRequest{Reference: "ref", ConnectorID: "id"}),
			Entry("amount missing", createPaymentRequest{Reference: "ref", ConnectorID: "id", CreatedAt: time.Now()}),
			Entry("payment type missing", createPaymentRequest{Reference: "ref", ConnectorID: "id", CreatedAt: time.Now(), Amount: big.NewInt(4467)}),
			Entry("payment type invalid", createPaymentRequest{Reference: "ref", ConnectorID: "id", CreatedAt: time.Now(), Amount: big.NewInt(4467), Type: "invalid"}),
			Entry("scheme missing", createPaymentRequest{Reference: "ref", ConnectorID: "id", CreatedAt: time.Now(), Amount: big.NewInt(4467), Type: "PAYOUT"}),
			Entry("scheme invalid", createPaymentRequest{Reference: "ref", ConnectorID: "id", CreatedAt: time.Now(), Amount: big.NewInt(4467), Type: "PAYOUT", Scheme: "invalid"}),
			Entry("asset missing", createPaymentRequest{Reference: "ref", ConnectorID: "id", CreatedAt: time.Now(), Amount: big.NewInt(4467), Type: "PAYOUT", Scheme: "CARD_VISA"}),
			Entry("adjustments missing", createPaymentRequest{Reference: "ref", ConnectorID: "id", CreatedAt: time.Now(), Amount: big.NewInt(4467), Type: "PAYOUT", Scheme: "CARD_VISA", Asset: "CAD"}),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("payment create err")
			m.EXPECT().PaymentsCreate(gomock.Any(), gomock.Any()).Return(expectedErr)
			cpr = createPaymentRequest{
				Reference:   "reference-err",
				ConnectorID: connID.String(),
				CreatedAt:   time.Now(),
				Amount:      big.NewInt(3500),
				Asset:       "JPY",
				Type:        models.PAYMENT_TYPE_PAYIN.String(),
				Scheme:      models.PAYMENT_SCHEME_CARD_AMEX.String(),
				Adjustments: adj,
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &cpr))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status created on success", func(ctx SpecContext) {
			m.EXPECT().PaymentsCreate(gomock.Any(), gomock.Any()).Return(nil)
			cpr = createPaymentRequest{
				Reference:   "reference-ok",
				ConnectorID: connID.String(),
				CreatedAt:   time.Now(),
				Amount:      big.NewInt(3500),
				Asset:       "JPY",
				Type:        models.PAYMENT_TYPE_PAYIN.String(),
				Scheme:      models.PAYMENT_SCHEME_CARD_AMEX.String(),
				Adjustments: adj,
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &cpr))
			assertExpectedResponse(w.Result(), http.StatusCreated, "data")
		})
	})
})