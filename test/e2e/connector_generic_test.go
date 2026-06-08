//go:build it

package test_suite

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	logging "github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	internalEvents "github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/client/models/components"
	"github.com/formancehq/payments/pkg/events"
	"github.com/formancehq/payments/pkg/testpsp"
	"github.com/formancehq/payments/pkg/testserver"
	"github.com/google/uuid"

	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Generic Connector E2E", Serial, func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()
	)

	app := testserver.NewTestServer(func() testserver.Configuration {
		return testserver.Configuration{
			Stack:                         stack,
			PostgresConfiguration:         db.GetValue().ConnectionOptions(),
			NatsURL:                       natsServer.GetValue().ClientURL(),
			TemporalNamespace:             temporalServer.GetValue().DefaultNamespace(),
			TemporalAddress:               temporalServer.GetValue().Address(),
			Output:                        GinkgoWriter,
			Debug:                         true,
			SkipOutboxScheduleCreation:    true,
			ConnectorPollingPeriodMinimum: 500 * time.Millisecond,
		}
	})

	AfterEach(func() {
		flushRemainingWorkflows(ctx)
	})

	When("installing a generic connector pointing at a test PSP server", Ordered, func() {
		var (
			pspServer   *testpsp.Server
			connectorID string
		)

		BeforeEach(func() {
			pspServer = testpsp.NewServer()
			DeferCleanup(pspServer.Close)

			install, err := app.GetValue().SDK().Payments.V3.InstallConnector(ctx, "generic",
				&components.V3InstallConnectorRequest{
					V3GenericConfig: &components.V3GenericConfig{
						APIKey:        "test-api-key",
						Endpoint:      pspServer.URL(),
						Name:          fmt.Sprintf("connector-%s", uuid.New()),
						PollingPeriod: pointer.For("2s"),
					},
				})
			Expect(err).To(BeNil())
			connectorID = install.V3InstallConnectorResponse.Data
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("calls psp side /accounts and emits account events", func() {
			expectedRefs := make([]any, len(pspServer.Accounts))
			for i, a := range pspServer.Accounts {
				expectedRefs[i] = HaveField("Reference", a.ID)
			}
			Eventually(func() []internalEvents.AccountMessagePayload {
				return loadAccountsByType(ctx, app.GetValue(), models.ACCOUNT_TYPE_INTERNAL)
			}).WithTimeout(10 * time.Second).WithPolling(500 * time.Millisecond).
				Should(ContainElements(expectedRefs...))
		})

		It("calls psp side /beneficiaries and emits external account events", func() {
			expectedRefs := make([]any, len(pspServer.Beneficiaries))
			for i, b := range pspServer.Beneficiaries {
				expectedRefs[i] = HaveField("Reference", b.ID)
			}
			Eventually(func() []internalEvents.AccountMessagePayload {
				return loadAccountsByType(ctx, app.GetValue(), models.ACCOUNT_TYPE_EXTERNAL)
			}).WithTimeout(10 * time.Second).WithPolling(500 * time.Millisecond).
				Should(ContainElements(expectedRefs...))
		})

		It("calls psp side /accounts/{id}/balances and emits balance events", func() {
			Eventually(func() int {
				n, _ := CountOutboxEventsByType(ctx, app.GetValue(), events.EventTypeSavedBalances)
				return n
			}).WithTimeout(10 * time.Second).WithPolling(time.Second).
				Should(Equal(len(pspServer.Accounts)))
		})

		It("can initiate and approve a payout via the PSP", func() {
			Eventually(func() int {
				return len(loadAccountsByType(ctx, app.GetValue(), models.ACCOUNT_TYPE_INTERNAL))
			}).WithTimeout(10 * time.Second).WithPolling(time.Second).
				Should(BeNumerically(">=", len(pspServer.Accounts)))

			Eventually(func() int {
				return len(loadAccountsByType(ctx, app.GetValue(), models.ACCOUNT_TYPE_EXTERNAL))
			}).WithTimeout(10 * time.Second).WithPolling(time.Second).
				Should(BeNumerically(">=", len(pspServer.Beneficiaries)))

			internalAccounts := loadAccountsByType(ctx, app.GetValue(), models.ACCOUNT_TYPE_INTERNAL)
			externalAccounts := loadAccountsByType(ctx, app.GetValue(), models.ACCOUNT_TYPE_EXTERNAL)

			initiateResp, err := app.GetValue().SDK().Payments.V3.InitiatePayment(ctx, pointer.For(false),
				&components.V3InitiatePaymentRequest{
					Reference:            uuid.New().String(),
					ConnectorID:          connectorID,
					Description:          "payout test",
					Type:                 components.V3PaymentInitiationTypeEnumPayout,
					Amount:               big.NewInt(1000),
					Asset:                "USD/2",
					SourceAccountID:      &internalAccounts[0].ID,
					DestinationAccountID: &externalAccounts[0].ID,
				})
			Expect(err).To(BeNil())
			Expect(initiateResp.V3InitiatePaymentResponse).NotTo(BeNil())
			paymentInitiationID := *initiateResp.GetV3InitiatePaymentResponse().Data.PaymentInitiationID

			approveResp, err := app.GetValue().SDK().Payments.V3.ApprovePaymentInitiation(ctx, paymentInitiationID)
			Expect(err).To(BeNil())
			Expect(approveResp.V3ApprovePaymentInitiationResponse).NotTo(BeNil())

			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			Eventually(taskPoller(approveResp.GetV3ApprovePaymentInitiationResponse().Data.TaskID)).
				WithTimeout(10 * time.Second).WithPolling(500 * time.Millisecond).
				Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))

			Expect(pspServer.PayoutsCalled()).To(BeNumerically(">", 0))

			resp, err := app.GetValue().SDK().Payments.V3.GetPaymentInitiation(ctx, paymentInitiationID)
			Expect(err).To(BeNil())
			Expect(resp.V3GetPaymentInitiationResponse).NotTo(BeNil())
			Expect(resp.V3GetPaymentInitiationResponse.Data.Status).To(Equal(components.V3PaymentInitiationStatusEnumProcessed))
		})

		It("can initiate and approve a transfer via the PSP", func() {
			Eventually(func() int {
				return len(loadAccountsByType(ctx, app.GetValue(), models.ACCOUNT_TYPE_INTERNAL))
			}).WithTimeout(10 * time.Second).WithPolling(time.Second).
				Should(BeNumerically(">=", len(pspServer.Accounts)))

			internalAccounts := loadAccountsByType(ctx, app.GetValue(), models.ACCOUNT_TYPE_INTERNAL)
			Expect(internalAccounts).To(HaveLen(len(pspServer.Accounts)))

			initiateResp, err := app.GetValue().SDK().Payments.V3.InitiatePayment(ctx, pointer.For(false),
				&components.V3InitiatePaymentRequest{
					Reference:            uuid.New().String(),
					ConnectorID:          connectorID,
					Description:          "transfer test",
					Type:                 components.V3PaymentInitiationTypeEnumTransfer,
					Amount:               big.NewInt(500),
					Asset:                "USD/2",
					SourceAccountID:      &internalAccounts[0].ID,
					DestinationAccountID: &internalAccounts[1].ID,
				})
			Expect(err).To(BeNil())
			Expect(initiateResp.V3InitiatePaymentResponse).NotTo(BeNil())
			paymentInitiationID := *initiateResp.GetV3InitiatePaymentResponse().Data.PaymentInitiationID

			approveResp, err := app.GetValue().SDK().Payments.V3.ApprovePaymentInitiation(ctx, paymentInitiationID)
			Expect(err).To(BeNil())
			Expect(approveResp.V3ApprovePaymentInitiationResponse).NotTo(BeNil())

			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			Eventually(taskPoller(approveResp.GetV3ApprovePaymentInitiationResponse().Data.TaskID)).
				WithTimeout(10 * time.Second).WithPolling(500 * time.Millisecond).
				Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))

			Expect(pspServer.TransfersCalled()).To(BeNumerically(">", 0))

			resp, err := app.GetValue().SDK().Payments.V3.GetPaymentInitiation(ctx, paymentInitiationID)
			Expect(err).To(BeNil())
			Expect(resp.V3GetPaymentInitiationResponse).NotTo(BeNil())
			Expect(resp.V3GetPaymentInitiationResponse.Data.Status).To(Equal(components.V3PaymentInitiationStatusEnumProcessed))
		})

		It("calls psp side /transactions and emits payment events", func() {
			expectedRefs := make([]any, len(pspServer.Transactions))
			for i, tx := range pspServer.Transactions {
				expectedRefs[i] = HaveField("Reference", tx.ID)
			}
			Eventually(func() []internalEvents.PaymentMessagePayload {
				payloads, err := LoadOutboxPayloadsByType(ctx, app.GetValue(), events.EventTypeSavedPayments)
				Expect(err).To(BeNil())
				msgs := make([]internalEvents.PaymentMessagePayload, 0, len(payloads))
				for _, p := range payloads {
					var msg internalEvents.PaymentMessagePayload
					Expect(json.Unmarshal(p, &msg)).To(Succeed())
					msgs = append(msgs, msg)
				}
				return msgs
			}).WithTimeout(10 * time.Second).WithPolling(500 * time.Millisecond).
				Should(ContainElements(expectedRefs...))
		})
	})
})

func loadAccountsByType(ctx context.Context, srv *Server, accountType models.AccountType) []internalEvents.AccountMessagePayload {
	GinkgoHelper()
	payloads, err := LoadOutboxPayloadsByType(ctx, srv, events.EventTypeSavedAccounts)
	Expect(err).ToNot(HaveOccurred(), "LoadOutboxPayloadsByType failed for accountType %s", accountType)
	accounts := make([]internalEvents.AccountMessagePayload, 0)
	for i, p := range payloads {
		var msg internalEvents.AccountMessagePayload
		Expect(json.Unmarshal(p, &msg)).ToNot(HaveOccurred(), "failed to unmarshal payload %d for accountType %s", i, accountType)
		if msg.Type == string(accountType) {
			accounts = append(accounts, msg)
		}
	}
	return accounts
}
