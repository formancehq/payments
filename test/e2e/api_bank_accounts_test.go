//go:build it

package test_suite

import (
	"fmt"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/testing/utils"
	v2 "github.com/formancehq/payments/internal/api/v2"
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

		accountNumber   = "123456789"
		iban            = "DE89370400440532013000"
		createRequest   v3.BankAccountsCreateRequest
		v2createRequest v2.BankAccountsCreateRequest

		app *utils.Deferred[*testserver.Server]
	)

	app = testserver.NewTestServer(func() Configuration {
		return Configuration{
			Stack:                 stack,
			PostgresConfiguration: db.GetValue().ConnectionOptions(),
			TemporalNamespace:     temporalServer.GetValue().DefaultNamespace(),
			TemporalAddress:       temporalServer.GetValue().Address(),
			Output:                GinkgoWriter,
		}
	})

	createRequest = v3.BankAccountsCreateRequest{
		Name:          "foo",
		AccountNumber: &accountNumber,
		IBAN:          &iban,
	}
	v2createRequest = v2.BankAccountsCreateRequest{
		Name:          "foo",
		AccountNumber: &accountNumber,
		IBAN:          &iban,
	}

	When("creating a new bank account with v3", func() {
		var (
			ver            int
			createResponse struct{ Data string }
			getResponse    struct{ Data models.BankAccount }
			err            error
		)
		JustBeforeEach(func() {
			ver = 3
			err = CreateBankAccount(ctx, app.GetValue(), ver, createRequest, &createResponse)
		})
		It("should be ok", func() {
			Expect(err).To(BeNil())
			id, err := uuid.Parse(createResponse.Data)
			Expect(err).To(BeNil())
			err = GetBankAccount(ctx, app.GetValue(), ver, id.String(), &getResponse)
			Expect(err).To(BeNil())
			Expect(getResponse.Data.ID.String()).To(Equal(id.String()))
		})
	})

	When("creating a new bank account with v2", func() {
		var (
			ver            int
			createResponse struct{ Data v2.BankAccountResponse }
			getResponse    struct{ Data models.BankAccount }
			err            error
		)
		JustBeforeEach(func() {
			ver = 2
			err = CreateBankAccount(ctx, app.GetValue(), ver, v2createRequest, &createResponse)
		})
		It("should be ok", func() {
			Expect(err).To(BeNil())
			id, err := uuid.Parse(createResponse.Data.ID)
			Expect(err).To(BeNil())
			err = GetBankAccount(ctx, app.GetValue(), ver, id.String(), &getResponse)
			Expect(err).To(BeNil())
			Expect(getResponse.Data.ID.String()).To(Equal(id.String()))
		})
	})

	When("forwarding a bank account to a connector with v3", func() {
		var (
			ver          int
			createRes    struct{ Data string }
			forwardReq   v3.BankAccountsForwardToConnectorRequest
			connectorRes struct{ Data string }
			res          struct{ Data models.Task }
			err          error
			id           uuid.UUID
		)
		JustBeforeEach(func() {
			ver = 3
			err = CreateBankAccount(ctx, app.GetValue(), ver, createRequest, &createRes)
			Expect(err).To(BeNil())
			id, err = uuid.Parse(createRes.Data)
			Expect(err).To(BeNil())

			connectorConf := ConnectorConf{
				Name:          fmt.Sprintf("connector-%s", id.String()),
				PollingPeriod: "2m",
				PageSize:      30,
				APIKey:        "key",
				Endpoint:      "http://example.com",
			}
			err := InstallConnector(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())
		})

		It("should fail when connector ID is invalid", func() {
			forwardReq = v3.BankAccountsForwardToConnectorRequest{ConnectorID: "invalid"}
			err = ForwardBankAccount(ctx, app.GetValue(), ver, id.String(), &forwardReq, &res)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("400"))
		})
		It("should be ok when connector is installed", func() {
			forwardReq = v3.BankAccountsForwardToConnectorRequest{ConnectorID: connectorRes.Data}
			err = ForwardBankAccount(ctx, app.GetValue(), ver, id.String(), &forwardReq, &res)
			Expect(err).To(BeNil())
			Expect(res.Data.ID.Reference).To(ContainSubstring(id.String()))
			Expect(res.Data.ID.Reference).To(ContainSubstring(connectorRes.Data))
			Expect(res.Data.ConnectorID.String()).To(ContainSubstring(connectorRes.Data))
		})
	})

	When("forwarding a bank account to a connector with v2", func() {
		var (
			ver          int
			createRes    struct{ Data v2.BankAccountResponse }
			forwardReq   v2.BankAccountsForwardToConnectorRequest
			connectorRes struct{ Data string }
			res          struct{ Data v2.BankAccountResponse }
			err          error
			id           uuid.UUID
		)
		JustBeforeEach(func() {
			ver = 2
			err = CreateBankAccount(ctx, app.GetValue(), ver, createRequest, &createRes)
			Expect(err).To(BeNil())
			id, err = uuid.Parse(createRes.Data.ID)
			Expect(err).To(BeNil())
			connectorConf := ConnectorConf{
				Name:          fmt.Sprintf("connector-%s", id.String()),
				PollingPeriod: "2m",
				PageSize:      30,
				APIKey:        "key",
				Endpoint:      "http://example.com",
			}
			err := InstallConnector(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())
		})
		It("should fail when connector ID is invalid", func() {
			forwardReq = v2.BankAccountsForwardToConnectorRequest{ConnectorID: "invalid"}
			err = ForwardBankAccount(ctx, app.GetValue(), ver, id.String(), &forwardReq, &res)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("400"))
		})
		It("should fail immediately with error because method is unimplemented on plugin", func() {
			forwardReq = v2.BankAccountsForwardToConnectorRequest{ConnectorID: connectorRes.Data}
			err = ForwardBankAccount(ctx, app.GetValue(), ver, id.String(), &forwardReq, &res)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("UNIMPLEMENTED"))
		})
	})

	When("updating bank account metadata with v3", func() {
		var (
			ver       int
			createRes struct{ Data string }
			res       struct{ Data models.BankAccount }
			req       v3.BankAccountsUpdateMetadataRequest
			err       error
			id        uuid.UUID
		)
		JustBeforeEach(func() {
			ver = 3
			err = CreateBankAccount(ctx, app.GetValue(), ver, createRequest, &createRes)
			Expect(err).To(BeNil())
			id, err = uuid.Parse(createRes.Data)
			Expect(err).To(BeNil())
		})

		It("should fail when metadata is invalid", func() {
			req = v3.BankAccountsUpdateMetadataRequest{}
			err = ForwardBankAccount(ctx, app.GetValue(), ver, id.String(), &req, &res)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("400"))
		})
		It("should be ok when metadata is valid", func() {
			req = v3.BankAccountsUpdateMetadataRequest{Metadata: map[string]string{"key": "val"}}
			err = UpdateBankAccountMetadata(ctx, app.GetValue(), ver, id.String(), &req, nil)
			Expect(err).To(BeNil())
			err = GetBankAccount(ctx, app.GetValue(), ver, id.String(), &res)
			Expect(err).To(BeNil())
			Expect(res.Data.ID.String()).To(Equal(id.String()))
			Expect(res.Data.Metadata).To(Equal(req.Metadata))
		})
	})

	When("updating bank account metadata with v2", func() {
		var (
			ver       int
			createRes struct{ Data v2.BankAccountResponse }
			res       struct{ Data models.BankAccount }
			req       v2.BankAccountsUpdateMetadataRequest
			err       error
			id        uuid.UUID
		)
		JustBeforeEach(func() {
			ver = 2
			err = CreateBankAccount(ctx, app.GetValue(), ver, createRequest, &createRes)
			Expect(err).To(BeNil())
			id, err = uuid.Parse(createRes.Data.ID)
			Expect(err).To(BeNil())
		})

		It("should fail when metadata is invalid", func() {
			req = v2.BankAccountsUpdateMetadataRequest{}
			err = ForwardBankAccount(ctx, app.GetValue(), ver, id.String(), &req, &res)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("400"))
		})
		It("should be ok when metadata is valid", func() {
			req = v2.BankAccountsUpdateMetadataRequest{Metadata: map[string]string{"key": "val"}}
			err = UpdateBankAccountMetadata(ctx, app.GetValue(), ver, id.String(), &req, nil)
			Expect(err).To(BeNil())
			err = GetBankAccount(ctx, app.GetValue(), ver, id.String(), &res)
			Expect(err).To(BeNil())
			Expect(res.Data.ID.String()).To(Equal(id.String()))
			Expect(res.Data.Metadata).To(Equal(req.Metadata))
		})
	})
})
