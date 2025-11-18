//go:build it

package test_suite

import (
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

var _ = Context("Payment API Payment Service Users", Ordered, Serial, func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()

		v3CreateRequest *components.V3CreatePaymentServiceUserRequest

		baID1 uuid.UUID
		baID2 uuid.UUID

		app *deferred.Deferred[*Server]
	)

	app = NewTestServer(func() Configuration {
		return Configuration{
			Stack:                     stack,
			NatsURL:                   natsServer.GetValue().ClientURL(),
			PostgresConfiguration:     db.GetValue().ConnectionOptions(),
			TemporalNamespace:         temporalServer.GetValue().DefaultNamespace(),
			TemporalAddress:           temporalServer.GetValue().Address(),
			Output:                    GinkgoWriter,
			SkipOutboxScheduleCreation: true,
		}
	})

	AfterEach(func() {
		flushRemainingWorkflows(ctx)
	})

	v3CreateRequest = &components.V3CreatePaymentServiceUserRequest{
		Name: "test",
		ContactDetails: &components.V3ContactDetailsRequest{
			Email:       pointer.For("dev@formance.com"),
			PhoneNumber: pointer.For("+33612131415"),
		},
		Address: &components.V3AddressRequest{
			StreetNumber: pointer.For("1"),
			StreetName:   pointer.For("test"),
			City:         pointer.For("test"),
			Region:       pointer.For("test"),
			PostalCode:   pointer.For("test"),
			Country:      pointer.For("FR"),
		},
		BankAccountIDs: []string{},
		Metadata:       map[string]string{},
	}

	BeforeEach(func() {
		createResponse, err := app.GetValue().SDK().Payments.V3.CreateBankAccount(ctx, &components.V3CreateBankAccountRequest{
			Name:          "foo",
			AccountNumber: pointer.For("123456789"),
			Iban:          pointer.For("DE89370400440532013000"),
			Country:       pointer.For("DE"),
		})
		Expect(err).To(BeNil())
		baID1, err = uuid.Parse(createResponse.GetV3CreateBankAccountResponse().Data)
		Expect(err).To(BeNil())

		createResponse, err = app.GetValue().SDK().Payments.V3.CreateBankAccount(ctx, &components.V3CreateBankAccountRequest{
			Name:          "bar",
			AccountNumber: pointer.For("123456789"),
			Iban:          pointer.For("DE89370400440532013000"),
			Country:       pointer.For("DE"),
		})
		Expect(err).To(BeNil())
		baID2, err = uuid.Parse(createResponse.GetV3CreateBankAccountResponse().Data)
		Expect(err).To(BeNil())

		// Only add the first bank account to the request, the second one will be added via the api
		v3CreateRequest.BankAccountIDs = []string{baID1.String()}
		_ = baID2

	})

	When("creating a payment service user", func() {
		var (
			psuID string
		)

		BeforeEach(func() {
			createResponse, err := app.GetValue().SDK().Payments.V3.CreatePaymentServiceUser(ctx, v3CreateRequest)
			Expect(err).To(BeNil())
			psuID = createResponse.GetV3CreatePaymentServiceUserResponse().Data
		})

		It("should be ok", func() {
			id, err := uuid.Parse(psuID)
			Expect(err).To(BeNil())

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPaymentServiceUser(ctx, psuID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetV3GetPaymentServiceUserResponse().Data.ID).To(Equal(id.String()))
		})
	})

	When("adding a bank account to a payment service user", func() {
		var (
			psuID string
		)

		BeforeEach(func() {
			createResponse, err := app.GetValue().SDK().Payments.V3.CreatePaymentServiceUser(ctx, v3CreateRequest)
			Expect(err).To(BeNil())
			psuID = createResponse.GetV3CreatePaymentServiceUserResponse().Data
		})

		It("should be ok", func() {
			_, err := app.GetValue().SDK().Payments.V3.AddBankAccountToPaymentServiceUser(ctx, psuID, baID2.String())
			Expect(err).To(BeNil())

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPaymentServiceUser(ctx, psuID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetV3GetPaymentServiceUserResponse().Data.BankAccountIDs).To(ContainElement(baID2.String()))
		})

		It("should not do anything if the bank account is already added", func() {
			_, err := app.GetValue().SDK().Payments.V3.AddBankAccountToPaymentServiceUser(ctx, psuID, baID2.String())
			Expect(err).To(BeNil())

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPaymentServiceUser(ctx, psuID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetV3GetPaymentServiceUserResponse().Data.BankAccountIDs).To(ContainElement(baID2.String()))

			_, err = app.GetValue().SDK().Payments.V3.AddBankAccountToPaymentServiceUser(ctx, psuID, baID2.String())
			Expect(err).To(BeNil())

			getResponse, err = app.GetValue().SDK().Payments.V3.GetPaymentServiceUser(ctx, psuID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetV3GetPaymentServiceUserResponse().Data.BankAccountIDs).To(ContainElement(baID2.String()))
		})

		It("should fail if bank account does not exists", func() {
			_, err := app.GetValue().SDK().Payments.V3.AddBankAccountToPaymentServiceUser(ctx, psuID, uuid.New().String())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to add bank account to payment service user: bank account: not found"))
		})

		It("should fail if payment service user does not exists", func() {
			_, err := app.GetValue().SDK().Payments.V3.AddBankAccountToPaymentServiceUser(ctx, uuid.New().String(), baID2.String())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to add bank account to payment service user: value not found"))
		})
	})

	When("forwarding a psu bank account to a connector", func() {
		var (
			connectorID string
			psuID       string
		)

		BeforeEach(func() {
			createResponse, err := app.GetValue().SDK().Payments.V3.CreatePaymentServiceUser(ctx, v3CreateRequest)
			Expect(err).To(BeNil())
			psuID = createResponse.GetV3CreatePaymentServiceUserResponse().Data

			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 3)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should fail when connector ID is invalid", func() {
			_, err := app.GetValue().SDK().Payments.V3.ForwardPaymentServiceUserBankAccount(ctx, psuID, baID1.String(), &components.V3ForwardPaymentServiceUserBankAccountRequest{
				ConnectorID: "invalid",
			})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("400"))
		})

		It("should fail when bank account ID is invalid", func() {
			_, err := app.GetValue().SDK().Payments.V3.ForwardPaymentServiceUserBankAccount(ctx, psuID, "invalid", &components.V3ForwardPaymentServiceUserBankAccountRequest{
				ConnectorID: connectorID,
			})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("400"))
		})

		It("should be ok when connector is installed", func() {
			beforeCount, err := CountOutboxEventsByType(ctx, app.GetValue(), events.EventTypeSavedBankAccount)
			Expect(err).To(BeNil())

			forwardResponse, err := app.GetValue().SDK().Payments.V3.ForwardPaymentServiceUserBankAccount(ctx, psuID, baID1.String(), &components.V3ForwardPaymentServiceUserBankAccountRequest{
				ConnectorID: connectorID,
			})
			Expect(err).To(BeNil())
			taskID, err := models.TaskIDFromString(forwardResponse.GetV3ForwardPaymentServiceUserBankAccountResponse().Data.TaskID)
			Expect(err).To(BeNil())
			Expect(taskID.Reference).To(ContainSubstring(baID1.String()))
			cID := models.MustConnectorIDFromString(connectorID)
			Expect(taskID.Reference).To(ContainSubstring(cID.Reference.String()))

			Eventually(func() (int, error) {
				return CountOutboxEventsByType(ctx, app.GetValue(), events.EventTypeSavedBankAccount)
			}).WithTimeout(3 * time.Second).Should(BeNumerically(">=", beforeCount+1))

			// Validate payload content from outbox_events
			// Fetch bank account to build expectations
			getResponse, err := app.GetValue().SDK().Payments.V3.GetBankAccount(ctx, baID1.String())
			Expect(err).To(BeNil())
			ba := getResponse.GetV3GetBankAccountResponse().Data

			payloads, err := LoadOutboxPayloadsByType(ctx, app.GetValue(), events.EventTypeSavedBankAccount)
			Expect(err).To(BeNil())

			var p internalEvents.BankAccountMessagePayload
			found := false
			for _, raw := range payloads {
				var tmp internalEvents.BankAccountMessagePayload
				Expect(json.Unmarshal(raw, &tmp)).To(Succeed())
				if tmp.ID == baID1.String() {
					p = tmp
					found = true
				}
			}
			Expect(found).To(BeTrue(), "expected a BANK_ACCOUNT_SAVED event payload for bank account %s", baID1.String())

			// Basic field expectations
			Expect(p.ID).To(Equal(baID1.String()))
			Expect(p.Country).To(Equal("DE"))
			Expect(p.Name).To(Equal(ba.Name))
			Expect(p.CreatedAt.Equal(ba.GetCreatedAt())).To(BeTrue())

			// Masking expectations: compute expected masked values from clear values returned by GET
			Expect(ba.AccountNumber).ToNot(BeNil())
			accountNumber := *ba.AccountNumber
			expectedMaskedAN := fmt.Sprintf("%s****%s", accountNumber[0:2], accountNumber[len(accountNumber)-3:])
			Expect(p.AccountNumber).To(Equal(expectedMaskedAN))

			Expect(ba.Iban).ToNot(BeNil())
			iban := *ba.Iban
			expectedMaskedIBAN := fmt.Sprintf("%s**************%s", iban[0:4], iban[len(iban)-4:])
			Expect(p.IBAN).To(Equal(expectedMaskedIBAN))

			// Metadata propagated from PSU
			Expect(p.Metadata).To(HaveKeyWithValue("com.formance.spec/owner/addressLine1", "1 test"))
			Expect(p.Metadata).To(HaveKeyWithValue("com.formance.spec/owner/city", "test"))
			Expect(p.Metadata).To(HaveKeyWithValue("com.formance.spec/owner/email", "dev@formance.com"))
			Expect(p.Metadata).To(HaveKeyWithValue("com.formance.spec/owner/phoneNumber", "+33612131415"))
			Expect(p.Metadata).To(HaveKeyWithValue("com.formance.spec/owner/postalCode", "test"))
			Expect(p.Metadata).To(HaveKeyWithValue("com.formance.spec/owner/region", "test"))
			Expect(p.Metadata).To(HaveKeyWithValue("com.formance.spec/owner/streetName", "test"))
			Expect(p.Metadata).To(HaveKeyWithValue("com.formance.spec/owner/streetNumber", "1"))

			// Related account created for the connector
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
			// AccountID is encoded; decode and validate fields
			accID, err := models.AccountIDFromString(ra.AccountID)
			Expect(err).To(BeNil())
			// AccountID is an encoded struct; validate its fields
			Expect(accID.Reference).To(Equal(fmt.Sprintf("dummypay-%s", baID1.String())))
			Expect(accID.ConnectorID.String()).To(Equal(connectorID))
			Expect(ra.CreatedAt.Equal(ba.GetCreatedAt())).To(BeTrue())
		})
	})

	When("deleting a payment service user", func() {
		var (
			psuID string
		)

		BeforeEach(func() {
			createResponse, err := app.GetValue().SDK().Payments.V3.CreatePaymentServiceUser(ctx, v3CreateRequest)
			Expect(err).To(BeNil())
			psuID = createResponse.GetV3CreatePaymentServiceUserResponse().Data
		})

		It("should be ok", func() {
			resp, err := app.GetValue().SDK().Payments.V3.DeletePaymentServiceUser(ctx, psuID)
			Expect(err).To(BeNil())
			Expect(resp.GetV3PaymentServiceUserDeleteResponse().Data).NotTo(BeNil())
			taskID, err := models.TaskIDFromString(resp.GetV3PaymentServiceUserDeleteResponse().Data.TaskID)
			Expect(err).To(BeNil())
			Expect(taskID.Reference).To(ContainSubstring("delete-user"))
		})

		It("should fail if payment service user does not exists", func() {
			_, err := app.GetValue().SDK().Payments.V3.DeletePaymentServiceUser(ctx, uuid.New().String())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot get payment service user"))
		})
	})
})
