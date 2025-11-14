package v3

import (
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"github.com/golang/mock/gomock"
)

var _ = Describe("API v3 Payment Initiation Reversal", func() {
	var (
		handlerFn http.HandlerFunc
		paymentID models.PaymentInitiationID
	)
	BeforeEach(func() {
		connID := models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		paymentID = models.PaymentInitiationID{Reference: "ref", ConnectorID: connID}
	})

	Context("reverse payment initiation", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentInitiationsReverse(m, validation.NewValidator())

			_ = paymentID
		})

		It("should return a bad request error when transferInitiationID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodPost, "paymentInitiationID", "invalid")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return a bad request error when body is missing", func(ctx SpecContext) {
			handlerFn(w, prepareQueryRequest(http.MethodGet, "paymentInitiationID", paymentID.String()))

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrMissingOrInvalidBody)
		})

		DescribeTable("validation errors",
			func(r PaymentInitiationsReverseRequest) {
				handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "paymentInitiationID", paymentID.String(), &r))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
			},
			Entry("reference missing", PaymentInitiationsReverseRequest{}),
			Entry("amount missing", PaymentInitiationsReverseRequest{Reference: "amount"}),
			Entry("asset missing", PaymentInitiationsReverseRequest{Reference: "asset", Amount: big.NewInt(1313)}),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("payment initiation reverse err")
			m.EXPECT().PaymentInitiationReversalsCreate(gomock.Any(), gomock.Any(), false).Return(
				models.Task{},
				expectedErr,
			)
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "paymentInitiationID", paymentID.String(), &PaymentInitiationsReverseRequest{
				Reference: "ref1",
				Amount:    big.NewInt(1313),
				Asset:     "USD/2",
			}))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status no content on success", func(ctx SpecContext) {
			m.EXPECT().PaymentInitiationReversalsCreate(gomock.Any(), gomock.Any(), false).Return(
				models.Task{},
				nil,
			)
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "paymentInitiationID", paymentID.String(), &PaymentInitiationsReverseRequest{
				Reference: "ref3",
				Amount:    big.NewInt(1313),
				Asset:     "eur/2",
			}))
			assertExpectedResponse(w.Result(), http.StatusAccepted, "")
		})
	})
})
