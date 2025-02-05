package v3

import (
	"bytes"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Payment Initiation Creation", func() {
	var (
		handlerFn http.HandlerFunc
		validate  *validator.Validate
		connID    models.ConnectorID
		source    models.AccountID
		dest      models.AccountID
		sourceID  string
		destID    string
		longText  string
	)
	BeforeEach(func() {
		validate = validation.NewValidator()

		connID = models.ConnectorID{Reference: uuid.New(), Provider: "dummypay"}
		source = models.AccountID{Reference: uuid.New().String(), ConnectorID: connID}
		dest = models.AccountID{Reference: uuid.New().String(), ConnectorID: connID}
		sourceID = source.String()
		destID = dest.String()

		var buf bytes.Buffer
		for i := 0; i < 10001; i++ {
			buf.WriteString("a")
		}
		longText = buf.String()
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
			handlerFn = paymentInitiationsCreate(m, validate)
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
			Entry("type missing", PaymentInitiationsCreateRequest{Reference: "type", ConnectorID: connID.String(), SourceAccountID: &sourceID, DestinationAccountID: &destID}),
			Entry("amount missing", PaymentInitiationsCreateRequest{Reference: "amount", SourceAccountID: &sourceID, DestinationAccountID: &destID, Type: "TRANSFER"}),
			Entry("asset missing", PaymentInitiationsCreateRequest{Reference: "asset", SourceAccountID: &sourceID, DestinationAccountID: &destID, Type: "TRANSFER", Amount: big.NewInt(1313)}),
			Entry("connectorID missing", PaymentInitiationsCreateRequest{Reference: "connector", SourceAccountID: &sourceID, DestinationAccountID: &destID, Type: "TRANSFER", Amount: big.NewInt(1717), Asset: "USD/2"}),
			Entry("reference too short", PaymentInitiationsCreateRequest{Reference: "qw", ConnectorID: connID.String(), SourceAccountID: &sourceID, DestinationAccountID: &destID, Type: "TRANSFER", Amount: big.NewInt(1717), Asset: "USD/2", ScheduledAt: time.Now().Add(time.Hour)}),
			Entry("reference too long", PaymentInitiationsCreateRequest{Reference: longText, ConnectorID: connID.String(), SourceAccountID: &sourceID, DestinationAccountID: &destID, Type: "TRANSFER", Amount: big.NewInt(1717), Asset: "USD/2", ScheduledAt: time.Now().Add(time.Hour)}),
			Entry("type is invalid", PaymentInitiationsCreateRequest{Reference: "type_invalid", ConnectorID: connID.String(), SourceAccountID: &sourceID, DestinationAccountID: &destID, Type: "SOMETYPE", Amount: big.NewInt(1717), Asset: "USD/2", ScheduledAt: time.Now().Add(time.Hour)}),
			Entry("asset is invalid", PaymentInitiationsCreateRequest{Reference: "asset_invalid", ConnectorID: connID.String(), SourceAccountID: &sourceID, DestinationAccountID: &destID, Type: "PAYOUT", Amount: big.NewInt(1717), Asset: "eur", ScheduledAt: time.Now().Add(time.Hour)}),
			Entry("connectorID is invalid", PaymentInitiationsCreateRequest{Reference: "connectorID_invalid", ConnectorID: "somestr", SourceAccountID: &sourceID, DestinationAccountID: &destID, Type: "PAYOUT", Amount: big.NewInt(1717), Asset: "eur/2", ScheduledAt: time.Now().Add(time.Hour)}),
			Entry("schedule is in the past", PaymentInitiationsCreateRequest{Reference: "schedule_is_past", ConnectorID: connID.String(), SourceAccountID: &sourceID, DestinationAccountID: &destID, Type: "TRANSFER", Amount: big.NewInt(1717), Asset: "USD/2", ScheduledAt: time.Now().Add(-time.Hour)}),
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
				Asset:                "EUR/2",
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &picr))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status accepted with only required fields", func(ctx SpecContext) {
			m.EXPECT().PaymentInitiationsCreate(gomock.Any(), gomock.Any(), false, false).Return(
				models.Task{},
				nil,
			)
			picr = PaymentInitiationsCreateRequest{
				Reference:            "ref-ok",
				ConnectorID:          connID.String(),
				DestinationAccountID: &destID,
				Type:                 "TRANSFER",
				Amount:               big.NewInt(2144),
				Asset:                "EUR/2",
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &picr))
			assertExpectedResponse(w.Result(), http.StatusAccepted, "data")
		})

		It("should return status accepted with all possible fields", func(ctx SpecContext) {
			m.EXPECT().PaymentInitiationsCreate(gomock.Any(), gomock.Any(), false, false).Return(
				models.Task{},
				nil,
			)
			picr = PaymentInitiationsCreateRequest{
				Reference:            "ref-ok",
				ScheduledAt:          time.Now().Add(time.Hour),
				ConnectorID:          connID.String(),
				Description:          "this is a payout",
				Type:                 "PAYOUT",
				Amount:               big.NewInt(45321),
				Asset:                "GBP/2",
				SourceAccountID:      &sourceID,
				DestinationAccountID: &destID,
				Metadata:             map[string]string{"meta": "data"},
			}
			handlerFn(w, prepareJSONRequest(http.MethodPost, &picr))
			assertExpectedResponse(w.Result(), http.StatusAccepted, "data")
		})
	})
})
