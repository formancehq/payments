package v3

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Payment Service Users Bank Accounts ForwardToConnector", func() {
	var (
		handlerFn     http.HandlerFunc
		bankAccountID uuid.UUID
		psuID         uuid.UUID
		connID        models.ConnectorID
	)
	BeforeEach(func() {
		psuID = uuid.New()
		bankAccountID = uuid.New()
		connID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
	})

	Context("forward psu bank accounts to connector", func() {
		var (
			w    *httptest.ResponseRecorder
			m    *backend.MockBackend
			freq PaymentServiceUserForwardBankAccountToConnectorRequest
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentServiceUsersForwardBankAccountToConnector(m, validation.NewValidator())
		})

		DescribeTable("validation errors",
			func(expected string, freq PaymentServiceUserForwardBankAccountToConnectorRequest, psuID string, baID string) {
				b, _ := json.Marshal(freq)
				body := bytes.NewReader(b)
				handlerFn(w, prepareQueryRequestWithBody(http.MethodPost, body, "paymentServiceUserID", psuID, "bankAccountID", baID))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, expected)
			},
			Entry("psu ID invalid", ErrInvalidID, PaymentServiceUserForwardBankAccountToConnectorRequest{}, "invalid", bankAccountID.String()),
			Entry("bank account ID ID invalid", ErrInvalidID, PaymentServiceUserForwardBankAccountToConnectorRequest{}, psuID.String(), "invalid"),
			Entry("connector ID missing", ErrValidation, PaymentServiceUserForwardBankAccountToConnectorRequest{}, psuID.String(), bankAccountID.String()),
			Entry("connector ID invalid", ErrValidation, PaymentServiceUserForwardBankAccountToConnectorRequest{ConnectorID: "blah"}, psuID.String(), bankAccountID.String()),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			m.EXPECT().PaymentServiceUsersForwardBankAccountToConnector(gomock.Any(), psuID, bankAccountID, connID).Return(
				models.Task{},
				fmt.Errorf("bank account forward err"),
			)
			freq = PaymentServiceUserForwardBankAccountToConnectorRequest{
				ConnectorID: connID.String(),
			}
			b, _ := json.Marshal(freq)
			body := bytes.NewReader(b)
			handlerFn(w, prepareQueryRequestWithBody(http.MethodPost, body, "paymentServiceUserID", psuID.String(), "bankAccountID", bankAccountID.String()))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status accepted on success", func(ctx SpecContext) {
			m.EXPECT().PaymentServiceUsersForwardBankAccountToConnector(gomock.Any(), psuID, bankAccountID, connID).Return(
				models.Task{},
				nil,
			)
			freq = PaymentServiceUserForwardBankAccountToConnectorRequest{
				ConnectorID: connID.String(),
			}
			b, _ := json.Marshal(freq)
			body := bytes.NewReader(b)
			handlerFn(w, prepareQueryRequestWithBody(http.MethodPost, body, "paymentServiceUserID", psuID.String(), "bankAccountID", bankAccountID.String()))
			assertExpectedResponse(w.Result(), http.StatusAccepted, "data")
		})
	})
})
