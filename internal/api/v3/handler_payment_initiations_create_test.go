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

var _ = Describe("API v3 Payment Initiation Creation", func() {
	var (
		handlerFn http.HandlerFunc
		connID    models.ConnectorID
		source    models.AccountID
		dest      models.AccountID
		sourceID  string
		destID    string
	)
	BeforeEach(func() {
		connID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		source = models.AccountID{Reference: uuid.New().String(), ConnectorID: connID}
		dest = models.AccountID{Reference: uuid.New().String(), ConnectorID: connID}
		sourceID = source.String()
		destID = dest.String()
	})

	Context("create payment initiation", func() {
		var (
			w    *httptest.ResponseRecorder
			m    *backend.MockBackend
			picr PaymentInitiationsCreateRequest
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentInitiationsCreate(m)
		})

		It("should return a bad request error when body is missing", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrMissingOrInvalidBody)
		})

		DescribeTable("validation errors",
			func(r PaymentInitiationsCreateRequest) {
				handlerFn(w, prepareJSONRequest(http.MethodPost, &r))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
			},
			Entry("reference missing", PaymentInitiationsCreateRequest{}),
			Entry("type missing", PaymentInitiationsCreateRequest{Reference: "type", SourceAccountID: &sourceID, DestinationAccountID: &destID}),
			Entry("amount missing", PaymentInitiationsCreateRequest{Reference: "amount", SourceAccountID: &sourceID, DestinationAccountID: &destID, Type: "TRANSFER"}),
			Entry("asset missing", PaymentInitiationsCreateRequest{Reference: "asset", SourceAccountID: &sourceID, DestinationAccountID: &destID, Type: "TRANSFER", Amount: big.NewInt(1313)}),
			Entry("connectorID missing", PaymentInitiationsCreateRequest{Reference: "connector", SourceAccountID: &sourceID, DestinationAccountID: &destID, Type: "TRANSFER", Amount: big.NewInt(1717), Asset: "USD"}),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("payment initiation create err")
			m.EXPECT().PaymentInitiationsCreate(gomock.Any(), gomock.Any(), false, false).Return(
				models.Task{},
				expectedErr,
			)
			picr = PaymentInitiationsCreateRequest{
				Reference:            "ref-err",
				ConnectorID:          connID.String(),
				SourceAccountID:      &sourceID,
				DestinationAccountID: &destID,
				Type:                 "TRANSFER",
				Amount:               big.NewInt(144),
				Asset:                "EUR",
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &picr))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status accepted on success", func(ctx SpecContext) {
			m.EXPECT().PaymentInitiationsCreate(gomock.Any(), gomock.Any(), false, false).Return(
				models.Task{},
				nil,
			)
			picr = PaymentInitiationsCreateRequest{
				Reference:            "ref-ok",
				ConnectorID:          connID.String(),
				SourceAccountID:      &sourceID,
				DestinationAccountID: &destID,
				Type:                 "TRANSFER",
				Amount:               big.NewInt(2144),
				Asset:                "EUR",
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &picr))
			assertExpectedResponse(w.Result(), http.StatusAccepted, "data")
		})
	})
})
