//go:build it

package test_suite

import (
	"context"
	"encoding/json"
	"fmt"
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
			ConnectorPollingPeriodMinimum: time.Second,
		}
	})

	AfterEach(func() {
		flushRemainingWorkflows(ctx)
	})

	When("installing a generic connector pointing at a test PSP server", func() {
		var (
			pspServer   *testpsp.Server
			connectorID string
		)

		BeforeEach(func() {
			pspServer = testpsp.NewServer()
			GinkgoT().Cleanup(pspServer.Close)

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
			}).WithTimeout(10 * time.Second).WithPolling(2 * time.Second).
				Should(BeNumerically(">=", len(pspServer.Accounts)))

			Expect(pspServer.AccountsCalled()).To(BeNumerically(">", 0))

			refs := loadAccountRefsByType(ctx, app.GetValue(), models.ACCOUNT_TYPE_INTERNAL)
			Expect(refs).To(HaveLen(len(pspServer.Accounts)))
			for _, account := range pspServer.Accounts {
				Expect(refs).To(ContainElement(account.ID))
			}
		})

		It("calls psp side /beneficiaries and emits external account events", func() {
			Eventually(func() int {
				return countAccountsByType(ctx, app.GetValue(), models.ACCOUNT_TYPE_EXTERNAL)
			}).WithTimeout(10 * time.Second).WithPolling(2 * time.Second).
				Should(BeNumerically(">=", len(pspServer.Beneficiaries)))

			Expect(pspServer.BeneficiariesCalled()).To(BeNumerically(">", 0))

			refs := loadAccountRefsByType(ctx, app.GetValue(), models.ACCOUNT_TYPE_EXTERNAL)
			Expect(refs).To(HaveLen(len(pspServer.Beneficiaries)))
			for _, ben := range pspServer.Beneficiaries {
				Expect(refs).To(ContainElement(ben.ID))
			}
		})

		It("calls psp side /accounts/{id}/balances and emits balance events", func() {
			Eventually(func() int {
				n, _ := CountOutboxEventsByType(ctx, app.GetValue(), events.EventTypeSavedBalances)
				return n
			}).WithTimeout(10 * time.Second).WithPolling(2 * time.Second).
				Should(BeNumerically(">=", len(pspServer.Accounts)))

			Expect(pspServer.BalancesCalled()).To(BeNumerically(">", 0))

			n, err := CountOutboxEventsByType(ctx, app.GetValue(), events.EventTypeSavedBalances)
			Expect(err).To(BeNil())
			Expect(n).To(Equal(len(pspServer.Accounts)))
		})

		It("calls psp side /transactions and emits payment events", func() {
			Eventually(func() int {
				n, _ := CountOutboxEventsByType(ctx, app.GetValue(), events.EventTypeSavedPayments)
				return n
			}).WithTimeout(10 * time.Second).WithPolling(2 * time.Second).
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
		})
	})
})

func countAccountsByType(ctx context.Context, srv *Server, accountType models.AccountType) int {
	return len(loadAccountRefsByType(ctx, srv, accountType))
}

func loadAccountRefsByType(ctx context.Context, srv *Server, accountType models.AccountType) []string {
	payloads, _ := LoadOutboxPayloadsByType(ctx, srv, events.EventTypeSavedAccounts)
	refs := make([]string, 0)
	for _, p := range payloads {
		var msg internalEvents.AccountMessagePayload
		if json.Unmarshal(p, &msg) == nil && msg.Type == string(accountType) {
			refs = append(refs, msg.Reference)
		}
	}
	return refs
}
