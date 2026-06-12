package v2

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
			handlerFn = bankAccountsForwardToConnector(m, validation.NewValidator())
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

		It("should obfuscate IBAN and account number in the response", func(ctx SpecContext) {
			const (
				fullIBAN          = "FR7630006000011234567890189"
				fullAccountNumber = "1234567890"
			)
			// Obfuscate mutates in place, so hand the backend its own copies to
			// avoid the handler altering the values we assert against.
			ibanForBackend := fullIBAN
			accountForBackend := fullAccountNumber
			m.EXPECT().BankAccountsForwardToConnector(gomock.Any(), bankAccountID, connID, true).Return(
				models.Task{},
				nil,
			)
			m.EXPECT().BankAccountsGet(gomock.Any(), bankAccountID).Return(
				&models.BankAccount{
					ID:            bankAccountID,
					Name:          "test",
					IBAN:          &ibanForBackend,
					AccountNumber: &accountForBackend,
				},
				nil,
			)
			freq = BankAccountsForwardToConnectorRequest{
				ConnectorID: connID.String(),
			}
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "bankAccountID", bankAccountID.String(), &freq))

			res := w.Result()
			defer res.Body.Close()
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			data, err := io.ReadAll(res.Body)
			Expect(err).To(BeNil())
			body := string(data)

			// Derive the expected masked values from Obfuscate itself so the
			// assertion can't drift from the masking implementation.
			ibanExpected := fullIBAN
			accountExpected := fullAccountNumber
			expected := &models.BankAccount{IBAN: &ibanExpected, AccountNumber: &accountExpected}
			Expect(expected.Obfuscate()).To(BeNil())
			Expect(body).To(ContainSubstring(ibanExpected))
			Expect(body).NotTo(ContainSubstring(fullIBAN))
			Expect(body).To(ContainSubstring(accountExpected))
			Expect(body).NotTo(ContainSubstring(fullAccountNumber))
		})
	})
})
