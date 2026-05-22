//go:build it

package test_suite

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
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
			Eventually(func() int {
				return countAccountsByType(ctx, app.GetValue(), models.ACCOUNT_TYPE_INTERNAL)
			}).WithTimeout(10 * time.Second).WithPolling(500 * time.Millisecond).
				Should(BeNumerically(">=", len(pspServer.Accounts)))

			Expect(pspServer.AccountsCalled()).To(BeNumerically(">", 0))

			accounts := loadAccountsByType(ctx, app.GetValue(), models.ACCOUNT_TYPE_INTERNAL)
			Expect(accounts).To(HaveLen(len(pspServer.Accounts)))
			for _, account := range pspServer.Accounts {
				Expect(accounts).To(ContainElement(HaveField("Reference", account.ID)))
			}

			// Trigger the schedule manually: the Temporal devserver's natural
			// schedule polling interval is much longer than the configured 2s,
			// so waiting for it would make this test slow and flaky.
			triggerConnectorSchedule(ctx, connectorID, "FETCH_ACCOUNTS")

			Eventually(func() int64 {
				return pspServer.AccountsCalled()
			}).WithTimeout(10 * time.Second).WithPolling(200 * time.Millisecond).
				Should(BeNumerically(">=", 2))

			latestAccount := pspServer.Accounts[len(pspServer.Accounts)-1]
			Expect(pspServer.LastSeenAccountPagingParamCreatedAtFrom()).To(Equal(latestAccount.CreatedAt))
		})

		It("calls psp side /beneficiaries and emits external account events", func() {
			Eventually(func() int {
				return countAccountsByType(ctx, app.GetValue(), models.ACCOUNT_TYPE_EXTERNAL)
			}).WithTimeout(10 * time.Second).WithPolling(500 * time.Millisecond).
				Should(BeNumerically(">=", len(pspServer.Beneficiaries)))

			Expect(pspServer.BeneficiariesCalled()).To(BeNumerically(">", 0))

			accounts := loadAccountsByType(ctx, app.GetValue(), models.ACCOUNT_TYPE_EXTERNAL)
			Expect(accounts).To(HaveLen(len(pspServer.Beneficiaries)))
			for _, ben := range pspServer.Beneficiaries {
				Expect(accounts).To(ContainElement(HaveField("Reference", ben.ID)))
			}

			triggerConnectorSchedule(ctx, connectorID, "FETCH_EXTERNAL_ACCOUNTS")

			Eventually(func() int64 {
				return pspServer.BeneficiariesCalled()
			}).WithTimeout(10 * time.Second).WithPolling(200 * time.Millisecond).
				Should(BeNumerically(">=", 2))

			latestBeneficiary := pspServer.Beneficiaries[len(pspServer.Beneficiaries)-1]
			Expect(pspServer.LastSeenBeneficiaryPagingParamCreatedAtFrom()).To(Equal(latestBeneficiary.CreatedAt))
		})

		It("calls psp side /accounts/{id}/balances and emits balance events", func() {
			Eventually(func() int {
				n, _ := CountOutboxEventsByType(ctx, app.GetValue(), events.EventTypeSavedBalances)
				return n
			}).WithTimeout(10 * time.Second).WithPolling(time.Second).
				Should(BeNumerically(">=", len(pspServer.Accounts)))

			Expect(pspServer.BalancesCalled()).To(BeNumerically(">", 0))

			n, err := CountOutboxEventsByType(ctx, app.GetValue(), events.EventTypeSavedBalances)
			Expect(err).To(BeNil())
			Expect(n).To(Equal(len(pspServer.Accounts)))
		})

		It("can initiate and approve a payout via the PSP", func() {
			Eventually(func() int {
				return countAccountsByType(ctx, app.GetValue(), models.ACCOUNT_TYPE_INTERNAL)
			}).WithTimeout(10 * time.Second).WithPolling(time.Second).
				Should(BeNumerically(">=", len(pspServer.Accounts)))

			Eventually(func() int {
				return countAccountsByType(ctx, app.GetValue(), models.ACCOUNT_TYPE_EXTERNAL)
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
			paymentInitiationID := *initiateResp.GetV3InitiatePaymentResponse().Data.PaymentInitiationID

			approveResp, err := app.GetValue().SDK().Payments.V3.ApprovePaymentInitiation(ctx, paymentInitiationID)
			Expect(err).To(BeNil())

			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			Eventually(taskPoller(approveResp.GetV3ApprovePaymentInitiationResponse().Data.TaskID)).
				WithTimeout(10 * time.Second).WithPolling(500 * time.Millisecond).
				Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))

			Expect(pspServer.PayoutsCalled()).To(BeNumerically(">", 0))
		})

		It("can initiate and approve a transfer via the PSP", func() {
			Eventually(func() int {
				return countAccountsByType(ctx, app.GetValue(), models.ACCOUNT_TYPE_INTERNAL)
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
			paymentInitiationID := *initiateResp.GetV3InitiatePaymentResponse().Data.PaymentInitiationID

			approveResp, err := app.GetValue().SDK().Payments.V3.ApprovePaymentInitiation(ctx, paymentInitiationID)
			Expect(err).To(BeNil())

			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			Eventually(taskPoller(approveResp.GetV3ApprovePaymentInitiationResponse().Data.TaskID)).
				WithTimeout(10 * time.Second).WithPolling(500 * time.Millisecond).
				Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))

			Expect(pspServer.TransfersCalled()).To(BeNumerically(">", 0))
		})

		It("calls psp side /transactions and emits payment events", func() {
			Eventually(func() int {
				n, _ := CountOutboxEventsByType(ctx, app.GetValue(), events.EventTypeSavedPayments)
				return n
			}).WithTimeout(10 * time.Second).WithPolling(500 * time.Millisecond).
				Should(BeNumerically(">=", len(pspServer.Transactions)))

			Expect(pspServer.TransactionsCalled()).To(BeNumerically(">", 0))

			payloads, err := LoadOutboxPayloadsByType(ctx, app.GetValue(), events.EventTypeSavedPayments)
			Expect(err).To(BeNil())
			Expect(payloads).To(HaveLen(len(pspServer.Transactions)))

			refs := make([]string, 0, len(payloads))
			for _, p := range payloads {
				var msg internalEvents.PaymentMessagePayload
				Expect(json.Unmarshal(p, &msg)).To(Succeed())
				refs = append(refs, msg.Reference)
			}
			for _, tx := range pspServer.Transactions {
				Expect(refs).To(ContainElement(tx.ID))
			}

			triggerConnectorSchedule(ctx, connectorID, "FETCH_PAYMENTS")

			Eventually(func() int64 {
				return pspServer.TransactionsCalled()
			}).WithTimeout(10 * time.Second).WithPolling(200 * time.Millisecond).
				Should(BeNumerically(">=", 2))

			latestTransaction := pspServer.Transactions[len(pspServer.Transactions)-1]
			Expect(pspServer.LastSeenTransactionPagingParamUpdatedAtFrom()).To(Equal(latestTransaction.UpdatedAt))
		})
	})
})

func loadAccountsByType(ctx context.Context, srv *Server, accountType models.AccountType) []internalEvents.AccountMessagePayload {
	payloads, _ := LoadOutboxPayloadsByType(ctx, srv, events.EventTypeSavedAccounts)
	accounts := make([]internalEvents.AccountMessagePayload, 0)
	for _, p := range payloads {
		var msg internalEvents.AccountMessagePayload
		if json.Unmarshal(p, &msg) == nil && msg.Type == string(accountType) {
			accounts = append(accounts, msg)
		}
	}
	return accounts
}

func countAccountsByType(ctx context.Context, srv *Server, accountType models.AccountType) int {
	return len(loadAccountsByType(ctx, srv, accountType))
}
