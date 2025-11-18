//go:build it

package test_suite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	internalEvents "github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/client/models/components"
	"github.com/formancehq/payments/pkg/events"
	"github.com/google/uuid"

	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Bank Accounts", Ordered, Serial, func() {
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
			Stack:                      stack,
			NatsURL:                    natsServer.GetValue().ClientURL(),
			PostgresConfiguration:      db.GetValue().ConnectionOptions(),
			TemporalNamespace:          temporalServer.GetValue().DefaultNamespace(),
			TemporalAddress:            temporalServer.GetValue().Address(),
			Output:                     GinkgoWriter,
			SkipOutboxScheduleCreation: true,
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
			id          uuid.UUID
		)

		BeforeEach(func() {
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

			p := waitSavedBankAccountPayloadForConnector(ctx, app.GetValue(), id.String(), connectorID)
			getResponse, err := app.GetValue().SDK().Payments.V3.GetBankAccount(ctx, id.String())
			Expect(err).To(BeNil())
			ba := getResponse.GetV3GetBankAccountResponse().Data
			assertSavedPayloadMatchesBankAccount(p, &ba, v3CreateRequest.Name, connectorID)
		})
	})

	When("forwarding a bank account to a connector with v2", func() {
		var (
			connectorID string
			id          uuid.UUID
		)
		BeforeEach(func() {
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
			_, err := app.GetValue().SDK().Payments.V1.ForwardBankAccount(ctx, id.String(), components.ForwardBankAccountRequest{
				ConnectorID: connectorID,
			})
			Expect(err).To(BeNil())

			p := waitSavedBankAccountPayloadForConnector(ctx, app.GetValue(), id.String(), connectorID)
			getResponse, err := app.GetValue().SDK().Payments.V1.GetBankAccount(ctx, id.String())
			Expect(err).To(BeNil())
			ba := getResponse.GetBankAccountResponse().Data
			assertSavedPayloadMatchesBankAccount(p, &ba, v2CreateRequest.Name, connectorID)
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

// test helpers shared across v2/v3 forwarding scenarios
func waitSavedBankAccountPayloadForConnector(ctx context.Context, s *Server, bankAccountID, connectorID string) internalEvents.BankAccountMessagePayload {
	var ret internalEvents.BankAccountMessagePayload
	Eventually(func(g Gomega) bool {
		payloads, err := LoadOutboxPayloadsByType(ctx, s, events.EventTypeSavedBankAccount)
		g.Expect(err).To(BeNil())
		for _, raw := range payloads {
			var tmp internalEvents.BankAccountMessagePayload
			if json.Unmarshal(raw, &tmp) == nil && tmp.ID == bankAccountID {
				for _, x := range tmp.RelatedAccounts {
					if x.ConnectorID == connectorID {
						ret = tmp
						return true
					}
				}
			}
		}
		return false
	}).WithTimeout(5 * time.Second).Should(BeTrue())
	return ret
}

// bankAccountLike abstracts both V1 and V3 bank account SDK models we need for assertions
// so we can share the same validation logic.
type bankAccountLike interface {
	GetID() string
	GetCreatedAt() time.Time
	GetAccountNumber() *string
	GetIban() *string
}

func assertSavedPayloadMatchesBankAccount(
	p internalEvents.BankAccountMessagePayload,
	ba bankAccountLike,
	expectedName, connectorID string,
) {
	Expect(p.ID).To(Equal(ba.GetID()))
	Expect(p.Country).To(Equal("DE"))
	Expect(p.Name).To(Equal(expectedName))
	Expect(p.CreatedAt.Equal(ba.GetCreatedAt())).To(BeTrue())

	Expect(ba.GetAccountNumber()).ToNot(BeNil())
	an := *ba.GetAccountNumber()
	Expect(p.AccountNumber).To(Equal(fmt.Sprintf("%s****%s", an[0:2], an[len(an)-3:])))

	Expect(ba.GetIban()).ToNot(BeNil())
	ib := *ba.GetIban()
	Expect(p.IBAN).To(Equal(fmt.Sprintf("%s**************%s", ib[0:4], ib[len(ib)-4:])))

	var ra internalEvents.BankAccountRelatedAccountsPayload
	raFound := false
	for _, x := range p.RelatedAccounts {
		if x.ConnectorID == connectorID {
			ra = x
			raFound = true
			break
		}
	}
	Expect(raFound).To(BeTrue(), "expected related account for connector %s", connectorID)
	Expect(ra.Provider).To(Equal("dummypay"))
	Expect(ra.ConnectorID).To(Equal(connectorID))
	accID, err := models.AccountIDFromString(ra.AccountID)
	Expect(err).To(BeNil())
	Expect(accID.Reference).To(Equal(fmt.Sprintf("dummypay-%s", ba.GetID())))
	Expect(accID.ConnectorID.String()).To(Equal(connectorID))
	Expect(ra.CreatedAt.Equal(ba.GetCreatedAt())).To(BeTrue())
}
