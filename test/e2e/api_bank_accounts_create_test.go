//go:build it

package test_suite

import (
	"github.com/formancehq/go-libs/v2/logging"
	v2 "github.com/formancehq/payments/internal/api/v2"
	v3 "github.com/formancehq/payments/internal/api/v3"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"

	. "github.com/formancehq/go-libs/v2/testing/utils"
	"github.com/formancehq/payments/pkg/testserver"
	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Bank Accounts", func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()

		temporalServer *testserver.TemporalServer
		app            *Deferred[*testserver.Server]

		accountNumber               = "123456789"
		iban                        = "DE89370400440532013000"
		bankAccountsCreateRequest   v3.BankAccountsCreateRequest
		bankAccountsV2CreateRequest v2.BankAccountsCreateRequest
	)
	temporalServer = testserver.CreateTemporalServer(GinkgoT())
	app = testserver.NewTestServer(func() Configuration {
		return Configuration{
			PostgresConfiguration: db.GetValue().ConnectionOptions(),
			TemporalNamespace:     temporalServer.GetDefaultNamespace(),
			TemporalAddress:       temporalServer.Address(),
			Output:                GinkgoWriter,
		}
	})

	BeforeEach(func() {
		bankAccountsCreateRequest = v3.BankAccountsCreateRequest{
			Name:          "foo",
			AccountNumber: &accountNumber,
			IBAN:          &iban,
		}
		bankAccountsV2CreateRequest = v2.BankAccountsCreateRequest{
			Name:          "foo",
			AccountNumber: &accountNumber,
			IBAN:          &iban,
		}
	})

	When("creating a new bank account with v3", func() {
		var (
			ver                        int
			bankAccountsCreateResponse struct{ Data string }
			bankAccountsGetResponse    models.BankAccount
			err                        error
		)
		JustBeforeEach(func() {
			ver = 3
			err = CreateBankAccount(ctx, app.GetValue(), ver, bankAccountsCreateRequest, &bankAccountsCreateResponse)
		})
		It("should be ok", func() {
			Expect(err).To(BeNil())
			id, err := uuid.Parse(bankAccountsCreateResponse.Data)
			Expect(err).To(BeNil())
			err = GetBankAccount(ctx, app.GetValue(), ver, id.String(), &bankAccountsGetResponse)
			Expect(err).To(BeNil())
		})
	})

	When("creating a new bank account with v2", func() {
		var (
			ver                        int
			bankAccountsCreateResponse struct{ Data v2.BankAccountResponse }
			bankAccountsGetResponse    models.BankAccount
			err                        error
		)
		JustBeforeEach(func() {
			ver = 2
			err = CreateBankAccount(ctx, app.GetValue(), ver, bankAccountsV2CreateRequest, &bankAccountsCreateResponse)
		})
		It("should be ok", func() {
			Expect(err).To(BeNil())
			id, err := uuid.Parse(bankAccountsCreateResponse.Data.ID)
			Expect(err).To(BeNil())
			err = GetBankAccount(ctx, app.GetValue(), ver, id.String(), &bankAccountsGetResponse)
			Expect(err).To(BeNil())
		})
	})

	When("forwarding a bank account to a connector with v3", func() {
		var (
			ver                        int
			bankAccountsCreateResponse struct{ Data string }
			forwardReq                 v3.BankAccountsForwardToConnectorRequest
			res                        struct{ Data models.Task }
			err                        error
			id                         uuid.UUID
		)
		JustBeforeEach(func() {
			ver = 3
			err = CreateBankAccount(ctx, app.GetValue(), ver, bankAccountsCreateRequest, &bankAccountsCreateResponse)
			Expect(err).To(BeNil())
			id, err = uuid.Parse(bankAccountsCreateResponse.Data)
			Expect(err).To(BeNil())
		})
		It("should should fail when connector ID is invalid", func() {
			forwardReq = v3.BankAccountsForwardToConnectorRequest{ConnectorID: "invalid"}
			err = ForwardBankAccount(ctx, app.GetValue(), ver, id.String(), &forwardReq, &res)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("400"))
		})
	})

	When("forwarding a bank account to a connector with v2", func() {
		var (
			ver                        int
			bankAccountsCreateResponse struct{ Data v2.BankAccountResponse }
			forwardReq                 v2.BankAccountsForwardToConnectorRequest
			res                        struct{ Data models.Task }
			err                        error
			id                         uuid.UUID
		)
		JustBeforeEach(func() {
			ver = 2
			err = CreateBankAccount(ctx, app.GetValue(), ver, bankAccountsCreateRequest, &bankAccountsCreateResponse)
			Expect(err).To(BeNil())
			id, err = uuid.Parse(bankAccountsCreateResponse.Data.ID)
			Expect(err).To(BeNil())
		})
		It("should should fail when connector ID is invalid", func() {
			forwardReq = v2.BankAccountsForwardToConnectorRequest{ConnectorID: "invalid"}
			err = ForwardBankAccount(ctx, app.GetValue(), ver, id.String(), &forwardReq, &res)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("400"))
		})
	})
})
