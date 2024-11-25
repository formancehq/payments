package v3

import (
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v2 Payment Initiation Reverse", func() {
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
			handlerFn = paymentInitiationsReverse(m)
		})

		It("should return a bad request error when transferInitiationID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "paymentInitiationID", "invalid")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return a bad request error when body is missing", func(ctx SpecContext) {
			handlerFn(w, prepareQueryRequest(http.MethodGet, "paymentInitiationID", paymentID.String()))

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrMissingOrInvalidBody)
		})

		DescribeTable("validation errors",
			func(r paymentInitiationsReverseRequest) {
				handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "paymentInitiationID", paymentID.String(), &r))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
			},
			Entry("reference missing", paymentInitiationsReverseRequest{}),
			Entry("amount missing", paymentInitiationsReverseRequest{Reference: "amount"}),
			Entry("asset missing", paymentInitiationsReverseRequest{Reference: "asset", Amount: big.NewInt(1313)}),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("payment initiation reverse err")
			m.EXPECT().PaymentInitiationReversalsCreate(gomock.Any(), gomock.Any(), true).Return(
				models.Task{},
				expectedErr,
			)
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "paymentInitiationID", paymentID.String(), &paymentInitiationsReverseRequest{
				Reference: "ref",
				Amount:    big.NewInt(1313),
				Asset:     "USD",
			}))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status no content on success", func(ctx SpecContext) {
			m.EXPECT().PaymentInitiationReversalsCreate(gomock.Any(), gomock.Any(), true).Return(
				models.Task{},
				nil,
			)
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "paymentInitiationID", paymentID.String(), &paymentInitiationsReverseRequest{
				Reference: "ref",
				Amount:    big.NewInt(1313),
				Asset:     "USD",
			}))
			assertExpectedResponse(w.Result(), http.StatusNoContent, "")
		})
	})
})
