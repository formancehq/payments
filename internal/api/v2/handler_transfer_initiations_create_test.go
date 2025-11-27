package v2

import (
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v2 Payment Initiation Creation", func() {
	var (
		handlerFn http.HandlerFunc
		validate  *validation.Validator
		connID    models.ConnectorID
		source    models.AccountID
		dest      models.AccountID
		sourceID  string
		destID    string
	)
	BeforeEach(func() {
		validate = validation.NewValidator()

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
			picr CreateTransferInitiationRequest
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = transferInitiationsCreate(m, validate)
		})

		It("should return a bad request error when body is missing", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrMissingOrInvalidBody)
		})

		DescribeTable("validation errors",
			func(r CreateTransferInitiationRequest) {
				handlerFn(w, prepareJSONRequest(http.MethodPost, &r))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
			},
			Entry("reference missing", CreateTransferInitiationRequest{}),
			Entry("type missing", CreateTransferInitiationRequest{Reference: "type", SourceAccountID: sourceID, DestinationAccountID: destID}),
			Entry("amount missing", CreateTransferInitiationRequest{Reference: "amount", SourceAccountID: sourceID, DestinationAccountID: destID, Type: "TRANSFER"}),
			Entry("asset missing", CreateTransferInitiationRequest{Reference: "asset", SourceAccountID: sourceID, DestinationAccountID: destID, Type: "TRANSFER", Amount: big.NewInt(1313)}),
			Entry("connectorID missing", CreateTransferInitiationRequest{Reference: "connector", SourceAccountID: sourceID, DestinationAccountID: destID, Type: "TRANSFER", Amount: big.NewInt(1717), Asset: "USD"}),
		)

		It("should return a CONFLICT error when entity already exists", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("already exists: %w", storage.ErrDuplicateKeyValue)
			m.EXPECT().PaymentInitiationsCreate(gomock.Any(), gomock.Any(), false, true).Return(
				models.Task{},
				expectedErr,
			)
			picr = CreateTransferInitiationRequest{
				Reference:            "ref-err",
				ConnectorID:          connID.String(),
				SourceAccountID:      sourceID,
				DestinationAccountID: destID,
				Type:                 "TRANSFER",
				Amount:               big.NewInt(144),
				Asset:                "EUR/2",
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &picr))
			assertExpectedResponse(w.Result(), http.StatusBadRequest, "CONFLICT")
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("payment initiation create err")
			m.EXPECT().PaymentInitiationsCreate(gomock.Any(), gomock.Any(), false, true).Return(
				models.Task{},
				expectedErr,
			)
			picr = CreateTransferInitiationRequest{
				Reference:            "ref-err",
				ConnectorID:          connID.String(),
				SourceAccountID:      sourceID,
				DestinationAccountID: destID,
				Type:                 "TRANSFER",
				Amount:               big.NewInt(144),
				Asset:                "EUR/2",
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
			picr = CreateTransferInitiationRequest{
				Reference:            "ref-ok",
				ConnectorID:          connID.String(),
				SourceAccountID:      sourceID,
				DestinationAccountID: destID,
				Type:                 "TRANSFER",
				Amount:               big.NewInt(2144),
				Asset:                "EUR/2",
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &picr))
			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
