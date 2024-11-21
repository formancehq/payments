package v2

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v2 Bank Accounts ForwardToConnector", func() {
	var (
		handlerFn     http.HandlerFunc
		bankAccountID uuid.UUID
		connID        models.ConnectorID
	)
	BeforeEach(func() {
		bankAccountID = uuid.New()
		connID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
	})

	Context("forward bank accounts to connector", func() {
		var (
			w    *httptest.ResponseRecorder
			m    *backend.MockBackend
			freq BankAccountsForwardToConnectorRequest
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = bankAccountsForwardToConnector(m)
		})

		DescribeTable("validation errors",
			func(expected string, freq BankAccountsForwardToConnectorRequest) {
				handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "bankAccountID", bankAccountID.String(), &freq))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, expected)
			},
			Entry("connector ID missing", ErrMissingOrInvalidBody, BankAccountsForwardToConnectorRequest{}),
			Entry("connector ID invalid", ErrValidation, BankAccountsForwardToConnectorRequest{ConnectorID: "blah"}),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			m.EXPECT().BankAccountsForwardToConnector(gomock.Any(), bankAccountID, connID, true).Return(
				models.Task{},
				fmt.Errorf("bank account forward err"),
			)
			freq = BankAccountsForwardToConnectorRequest{
				ConnectorID: connID.String(),
			}
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "bankAccountID", bankAccountID.String(), &freq))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status accepted on success", func(ctx SpecContext) {
			m.EXPECT().BankAccountsForwardToConnector(gomock.Any(), bankAccountID, connID, true).Return(
				models.Task{},
				nil,
			)
			m.EXPECT().BankAccountsGet(gomock.Any(), bankAccountID).Return(
				&models.BankAccount{},
				nil,
			)
			freq = BankAccountsForwardToConnectorRequest{
				ConnectorID: connID.String(),
			}
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "bankAccountID", bankAccountID.String(), &freq))
			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
