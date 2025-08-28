//go:build it

package test_suite

import (
	"fmt"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/client/models/components"
	evts "github.com/formancehq/payments/pkg/events"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Bank Accounts", Serial, func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()

		accountNumber   = "123456789"
		iban            = "DE89370400440532013000"
		v3CreateRequest *components.V3CreateBankAccountRequest
		v2CreateRequest components.BankAccountRequest

		app *deferred.Deferred[*Server]
	)

	app = NewTestServer(func() Configuration {
		return Configuration{
			Stack:                 stack,
			NatsURL:               natsServer.GetValue().ClientURL(),
			PostgresConfiguration: db.GetValue().ConnectionOptions(),
			TemporalNamespace:     temporalServer.GetValue().DefaultNamespace(),
			TemporalAddress:       temporalServer.GetValue().Address(),
			Output:                GinkgoWriter,
		}
	})

	AfterEach(func() {
		flushRemainingWorkflows(ctx)
	})

	v3CreateRequest = &components.V3CreateBankAccountRequest{
		Name:          "foo",
		AccountNumber: &accountNumber,
		Iban:          &iban,
		Country:       pointer.For("DE"),
	}

	v2CreateRequest = components.BankAccountRequest{
		Name:          "foo",
		AccountNumber: &accountNumber,
		Iban:          &iban,
		Country:       "DE",
	}

	When("creating a new bank account with v3", func() {
		var (
			bankAccountID string
		)
		BeforeEach(func() {
			createResponse, err := app.GetValue().SDK().Payments.V3.CreateBankAccount(ctx, v3CreateRequest)
			Expect(err).To(BeNil())
			bankAccountID = createResponse.GetV3CreateBankAccountResponse().Data
		})

		It("should be ok", func() {
			id, err := uuid.Parse(bankAccountID)
			Expect(err).To(BeNil())

			getResponse, err := app.GetValue().SDK().Payments.V3.GetBankAccount(ctx, bankAccountID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetV3GetBankAccountResponse().Data.ID).To(Equal(id.String()))
		})
	})

	When("creating a new bank account with v2", func() {
		var (
			bankAccountID string
		)
		BeforeEach(func() {
			createResponse, err := app.GetValue().SDK().Payments.V1.CreateBankAccount(ctx, v2CreateRequest)
			Expect(err).To(BeNil())
			bankAccountID = createResponse.GetBankAccountResponse().Data.ID
		})
		It("should be ok", func() {
			id, err := uuid.Parse(bankAccountID)
			Expect(err).To(BeNil())

			getResponse, err := app.GetValue().SDK().Payments.V1.GetBankAccount(ctx, bankAccountID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetBankAccountResponse().Data.ID).To(Equal(id.String()))
		})
	})

	When("forwarding a bank account to a connector with v3", func() {
		var (
			connectorID string
			e           chan *nats.Msg
			id          uuid.UUID
		)

		BeforeEach(func() {
			e = Subscribe(GinkgoT(), app.GetValue())

			createResponse, err := app.GetValue().SDK().Payments.V3.CreateBankAccount(ctx, v3CreateRequest)
			Expect(err).To(BeNil())
			id, err = uuid.Parse(createResponse.GetV3CreateBankAccountResponse().Data)
			Expect(err).To(BeNil())

			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 3)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should fail when connector ID is invalid", func() {
			_, err := app.GetValue().SDK().Payments.V3.ForwardBankAccount(ctx, id.String(), &components.V3ForwardBankAccountRequest{
				ConnectorID: "invalid",
			})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("400"))
		})
		It("should be ok when connector is installed", func() {
			forwardResponse, err := app.GetValue().SDK().Payments.V3.ForwardBankAccount(ctx, id.String(), &components.V3ForwardBankAccountRequest{
				ConnectorID: connectorID,
			})
			Expect(err).To(BeNil())
			taskID, err := models.TaskIDFromString(forwardResponse.GetV3ForwardBankAccountResponse().Data.TaskID)
			Expect(err).To(BeNil())
			Expect(taskID.Reference).To(ContainSubstring(id.String()))
			cID := models.MustConnectorIDFromString(connectorID)
			Expect(taskID.Reference).To(ContainSubstring(cID.Reference.String()))

			connectorID, err := models.ConnectorIDFromString(connectorID)
			Expect(err).To(BeNil())

			getResponse, err := app.GetValue().SDK().Payments.V3.GetBankAccount(ctx, id.String())
			Expect(err).To(BeNil())

			Expect(getResponse.GetV3GetBankAccountResponse().Data.AccountNumber).ToNot(BeNil())
			accountNumber := *getResponse.GetV3GetBankAccountResponse().Data.AccountNumber
			Expect(getResponse.GetV3GetBankAccountResponse().Data.Iban).ToNot(BeNil())
			iban := *getResponse.GetV3GetBankAccountResponse().Data.Iban

			accountID := models.AccountID{
				Reference:   fmt.Sprintf("dummypay-%s", id.String()),
				ConnectorID: connectorID,
			}

			Eventually(e).Should(Receive(Event(evts.EventTypeSavedBankAccount, WithPayload(
				events.BankAccountMessagePayload{
					ID:            id.String(),
					Country:       "DE",
					Name:          v3CreateRequest.Name,
					AccountNumber: fmt.Sprintf("%s****%s", accountNumber[0:2], accountNumber[len(accountNumber)-3:]),
					IBAN:          fmt.Sprintf("%s**************%s", iban[0:4], iban[len(iban)-4:]),
					CreatedAt:     getResponse.GetV3GetBankAccountResponse().Data.GetCreatedAt(),
					RelatedAccounts: []events.BankAccountRelatedAccountsPayload{
						{
							AccountID:   accountID.String(),
							CreatedAt:   getResponse.GetV3GetBankAccountResponse().Data.GetCreatedAt(),
							ConnectorID: connectorID.String(),
							Provider:    "dummypay",
						},
					},
				},
			))))
		})
	})

	When("forwarding a bank account to a connector with v2", func() {
		var (
			connectorID string
			e           chan *nats.Msg
			id          uuid.UUID
		)
		BeforeEach(func() {
			e = Subscribe(GinkgoT(), app.GetValue())

			createResponse, err := app.GetValue().SDK().Payments.V1.CreateBankAccount(ctx, v2CreateRequest)
			Expect(err).To(BeNil())
			id, err = uuid.Parse(createResponse.GetBankAccountResponse().Data.ID)
			Expect(err).To(BeNil())
			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 2)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should fail when connector ID is invalid", func() {
			_, err := app.GetValue().SDK().Payments.V1.ForwardBankAccount(ctx, id.String(), components.ForwardBankAccountRequest{
				ConnectorID: "invalid",
			})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("400"))
		})
		It("should be ok", func() {
			forwardResponse, err := app.GetValue().SDK().Payments.V1.ForwardBankAccount(ctx, id.String(), components.ForwardBankAccountRequest{
				ConnectorID: connectorID,
			})
			Expect(err).To(BeNil())
			Expect(forwardResponse.GetBankAccountResponse().Data.RelatedAccounts).To(HaveLen(1))
			Expect(forwardResponse.GetBankAccountResponse().Data.RelatedAccounts[0].ConnectorID).To(Equal(connectorID))

			Eventually(e).Should(Receive(Event(evts.EventTypeSavedBankAccount)))
		})
	})

	When("updating bank account metadata with v3", func() {
		var (
			id uuid.UUID
		)
		BeforeEach(func() {
			createResponse, err := app.GetValue().SDK().Payments.V3.CreateBankAccount(ctx, v3CreateRequest)
			Expect(err).To(BeNil())
			id, err = uuid.Parse(createResponse.GetV3CreateBankAccountResponse().Data)
			Expect(err).To(BeNil())
		})

		It("should fail when metadata is invalid", func() {
			_, err := app.GetValue().SDK().Payments.V3.UpdateBankAccountMetadata(ctx, id.String(), &components.V3UpdateBankAccountMetadataRequest{})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("400"))
		})
		It("should be ok when metadata is valid", func() {
			metadata := map[string]string{"key": "val"}
			_, err := app.GetValue().SDK().Payments.V3.UpdateBankAccountMetadata(ctx, id.String(), &components.V3UpdateBankAccountMetadataRequest{
				Metadata: metadata,
			})
			Expect(err).To(BeNil())

			getResponse, err := app.GetValue().SDK().Payments.V3.GetBankAccount(ctx, id.String())
			Expect(err).To(BeNil())
			Expect(getResponse.GetV3GetBankAccountResponse().Data.ID).To(Equal(id.String()))
			Expect(getResponse.GetV3GetBankAccountResponse().Data.Metadata).To(Equal(metadata))
		})
	})

	When("updating bank account metadata with v2", func() {
		var (
			id uuid.UUID
		)
		BeforeEach(func() {
			createResponse, err := app.GetValue().SDK().Payments.V1.CreateBankAccount(ctx, v2CreateRequest)
			Expect(err).To(BeNil())
			id, err = uuid.Parse(createResponse.GetBankAccountResponse().Data.ID)
			Expect(err).To(BeNil())
		})

		It("should fail when metadata is invalid", func() {
			_, err := app.GetValue().SDK().Payments.V1.UpdateBankAccountMetadata(ctx, id.String(), components.UpdateBankAccountMetadataRequest{})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("400"))
		})
		It("should be ok when metadata is valid", func() {
			metadata := map[string]string{"key": "val"}
			_, err := app.GetValue().SDK().Payments.V1.UpdateBankAccountMetadata(ctx, id.String(), components.UpdateBankAccountMetadataRequest{
				Metadata: metadata,
			})
			Expect(err).To(BeNil())

			getResponse, err := app.GetValue().SDK().Payments.V1.GetBankAccount(ctx, id.String())
			Expect(err).To(BeNil())
			Expect(getResponse.GetBankAccountResponse().Data.ID).To(Equal(id.String()))
			Expect(getResponse.GetBankAccountResponse().Data.Metadata).To(Equal(metadata))
		})
	})
})
