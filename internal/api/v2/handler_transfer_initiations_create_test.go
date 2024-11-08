package v2

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

var _ = Describe("API v2 Payment Initiation Creation", func() {
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
			picr createTransferInitiationRequest
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = transferInitiationsCreate(m)
		})

		It("should return a bad request error when body is missing", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrMissingOrInvalidBody)
		})

		DescribeTable("validation errors",
			func(r createTransferInitiationRequest) {
				handlerFn(w, prepareJSONRequest(http.MethodPost, &r))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
			},
			Entry("reference missing", createTransferInitiationRequest{}),
			Entry("type missing", createTransferInitiationRequest{Reference: "type", SourceAccountID: sourceID, DestinationAccountID: destID}),
			Entry("amount missing", createTransferInitiationRequest{Reference: "amount", SourceAccountID: sourceID, DestinationAccountID: destID, Type: "TRANSFER"}),
			Entry("asset missing", createTransferInitiationRequest{Reference: "asset", SourceAccountID: sourceID, DestinationAccountID: destID, Type: "TRANSFER", Amount: big.NewInt(1313)}),
			Entry("connectorID missing", createTransferInitiationRequest{Reference: "connector", SourceAccountID: sourceID, DestinationAccountID: destID, Type: "TRANSFER", Amount: big.NewInt(1717), Asset: "USD"}),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("payment initiation create err")
			m.EXPECT().PaymentInitiationsCreate(gomock.Any(), gomock.Any(), false, true).Return(
				models.Task{},
				expectedErr,
			)
			picr = createTransferInitiationRequest{
				Reference:            "ref-err",
				ConnectorID:          connID.String(),
				SourceAccountID:      sourceID,
				DestinationAccountID: destID,
				Type:                 "TRANSFER",
				Amount:               big.NewInt(144),
				Asset:                "EUR",
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &picr))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status ok on success", func(ctx SpecContext) {
			m.EXPECT().PaymentInitiationsCreate(gomock.Any(), gomock.Any(), false, true).Return(
				models.Task{},
				nil,
			)
			m.EXPECT().PaymentInitiationAdjustmentsGetLast(gomock.Any(), gomock.Any()).Return(
				&models.PaymentInitiationAdjustment{},
				nil,
			)
			picr = createTransferInitiationRequest{
				Reference:            "ref-ok",
				ConnectorID:          connID.String(),
				SourceAccountID:      sourceID,
				DestinationAccountID: destID,
				Type:                 "TRANSFER",
				Amount:               big.NewInt(2144),
				Asset:                "EUR",
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &picr))
			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})