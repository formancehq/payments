//go:build it

package test_suite

import (
	"github.com/formancehq/go-libs/v2/logging"
	v3 "github.com/formancehq/payments/internal/api/v3"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"

	"github.com/formancehq/payments/pkg/testserver"
	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Bank Accounts", func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()

		temporalServer = testserver.CreateTemporalServer(GinkgoT())

		accountNumber = "123456789"
		iban          = "DE89370400440532013000"
	)

	server := testserver.NewTestServer(func() Configuration {
		return Configuration{
			PostgresConfiguration: db.GetValue().ConnectionOptions(),
			TemporalNamespace:     temporalServer.GetDefaultNamespace(),
			TemporalAddress:       temporalServer.Address(),
			Output:                GinkgoWriter,
		}
	})
	When("creating a new bank account with v3", func() {
		var (
			bankAccountsCreateRequest  v3.BankAccountsCreateRequest
			bankAccountsCreateResponse struct{ Data string }
			bankAccountsGetResponse    models.BankAccount
			err                        error
		)
		BeforeEach(func() {
			bankAccountsCreateRequest = v3.BankAccountsCreateRequest{
				Name:          "foo",
				AccountNumber: &accountNumber,
				IBAN:          &iban,
			}
		})
		JustBeforeEach(func() {
			err = CreateBankAccount(ctx, server.GetValue(), bankAccountsCreateRequest, &bankAccountsCreateResponse)
		})
		It("should be ok", func() {
			Expect(err).To(BeNil())
			id, err := uuid.Parse(bankAccountsCreateResponse.Data)
			Expect(err).To(BeNil())
			err = GetBankAccount(ctx, server.GetValue(), id.String(), &bankAccountsGetResponse)
			Expect(err).To(BeNil())
		})
	})
})
