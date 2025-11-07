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
	"github.com/formancehq/go-libs/v3/testing/deferred"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/client/models/components"
	"github.com/formancehq/payments/pkg/client/models/operations"
	"github.com/formancehq/payments/pkg/testserver"
	. "github.com/formancehq/payments/pkg/testserver"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Accounts", Serial, func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()

		app *deferred.Deferred[*Server]
	)

	app = NewTestServer(func() Configuration {
		return Configuration{
			Stack:                 stack,
			PostgresConfiguration: db.GetValue().ConnectionOptions(),
			NatsURL:               natsServer.GetValue().ClientURL(),
			TemporalNamespace:     temporalServer.GetValue().DefaultNamespace(),
			TemporalAddress:       temporalServer.GetValue().Address(),
			Output:                GinkgoWriter,
		}
	})

	AfterEach(func() {
		flushRemainingWorkflows(ctx)
	})

	createdAt, _ := time.Parse("2006-Jan-02", "2024-Nov-29")

	When("creating a new account", func() {
		var (
			connectorUUID uuid.UUID
			connectorID   string
			err           error
		)

		BeforeEach(func() {
			connectorUUID = uuid.New()
			connectorID, err = installConnector(ctx, app.GetValue(), connectorUUID, 3)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should be successful with v2", func() {
			createResponse, err := app.GetValue().SDK().Payments.V1.CreateAccount(ctx, components.AccountRequest{
				Reference:    "ref",
				ConnectorID:  connectorID,
				CreatedAt:    createdAt,
				Type:         "INTERNAL",
				DefaultAsset: pointer.For("USD/2"),
				AccountName:  pointer.For("foo"),
				Metadata: map[string]string{
					"key": "val",
				},
			})
			Expect(err).To(BeNil())

			getResponse, err := app.GetValue().SDK().Payments.V1.GetAccount(ctx, createResponse.GetAccountResponse().Data.ID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetAccountResponse().Data).To(Equal(createResponse.GetAccountResponse().Data))

			Eventually(app.GetValue()).Should(OutboxEvent(models.OUTBOX_EVENT_ACCOUNT_SAVED))
		})

		It("should be successful with v3", func() {
			createResponse, err := app.GetValue().SDK().Payments.V3.CreateAccount(ctx, &components.V3CreateAccountRequest{
				Reference:    "ref",
				ConnectorID:  connectorID,
				CreatedAt:    createdAt,
				AccountName:  "foo",
				Type:         "INTERNAL",
				DefaultAsset: pointer.For("USD/2"),
				Metadata: map[string]string{
					"key": "val",
				},
			})
			Expect(err).To(BeNil())

			getResponse, err := app.GetValue().SDK().Payments.V3.GetAccount(ctx, createResponse.GetV3CreateAccountResponse().Data.ID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetV3GetAccountResponse().Data).To(Equal(createResponse.GetV3CreateAccountResponse().Data))
			Expect(getResponse.GetV3GetAccountResponse().Data.Connector).NotTo(BeNil())
			Expect(createResponse.GetV3CreateAccountResponse().Data.Connector).NotTo(BeNil())

			Expect(createResponse.GetV3CreateAccountResponse().Data.Connector.Name).NotTo(BeNil())
			Expect(*createResponse.GetV3CreateAccountResponse().Data.Connector.Name).To(ContainSubstring(connectorUUID.String()))

			Eventually(app.GetValue()).Should(OutboxEvent(models.OUTBOX_EVENT_ACCOUNT_SAVED))
		})

		It("temporarily verifies an outbox_event is created on account creation", func() {
			// Create an account and then check the outbox_events table for the corresponding event
			createResponse, err := app.GetValue().SDK().Payments.V3.CreateAccount(ctx, &components.V3CreateAccountRequest{
				Reference:    "ref-outbox",
				ConnectorID:  connectorID,
				CreatedAt:    createdAt,
				AccountName:  "foo-outbox",
				Type:         "INTERNAL",
				DefaultAsset: pointer.For("USD/2"),
				Metadata:     map[string]string{},
			})
			Expect(err).To(BeNil())
			accountID := createResponse.GetV3CreateAccountResponse().Data.ID

			// Access the database and assert an outbox event exists for this account
			Eventually(func(g Gomega) {
				db, err := app.GetValue().Database()
				g.Expect(err).To(BeNil())
				defer db.Close()

				count, err := db.NewSelect().
					TableExpr("outbox_events").
					Where("event_type = ? AND entity_id = ?", models.OUTBOX_EVENT_ACCOUNT_SAVED, accountID).
					Count(ctx)
				g.Expect(err).To(BeNil())
				g.Expect(count).To(BeNumerically(">", 0))
			}).WithTimeout(10 * time.Second).Should(Succeed())

			Eventually(func(g Gomega) {
				db, err := app.GetValue().Database()
				defer db.Close()
				count, err := db.NewSelect().
					TableExpr("outbox_events").
					Where("event_type = ? AND entity_id = ?", models.OUTBOX_EVENT_ACCOUNT_SAVED, accountID).
					Count(ctx)
				g.Expect(err).To(BeNil())
				g.Expect(count).To(BeNumerically("==", 0))
			}).WithTimeout(10 * time.Second).Should(Succeed())

		})
	})

	When("fetching account balances", func() {
		var (
			connectorID string
			err         error
		)

		BeforeEach(func() {
			id := uuid.New()
			connectorConf := newV3ConnectorConfigFn()(id)
			connectorID, err = installV3Connector(ctx, app.GetValue(), connectorConf, uuid.New())
			Expect(err).To(BeNil())
			_, err = GeneratePSPData(connectorConf.Directory)
			Expect(err).To(BeNil())
			Eventually(app.GetValue()).Should(OutboxEvent(models.OUTBOX_EVENT_ACCOUNT_SAVED))
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should be successful with v2", func() {
			var msg events.BalanceMessagePayload
			Eventually(app.GetValue()).
				Should(OutboxEvent(models.OUTBOX_EVENT_BALANCE_SAVED, WithRawCallback(func(b []byte) error {
					// Outbox payload uses balance as a string; adapt to BalanceMessagePayload
					type outboxBalance struct {
						AccountID     string    `json:"accountID"`
						ConnectorID   string    `json:"connectorID"`
						Provider      string    `json:"provider"`
						CreatedAt     time.Time `json:"createdAt"`
						LastUpdatedAt time.Time `json:"lastUpdatedAt"`
						Asset         string    `json:"asset"`
						Balance       string    `json:"balance"`
					}
					var tmp outboxBalance
					if err := json.Unmarshal(b, &tmp); err != nil {
						return err
					}
					bi, ok := new(big.Int).SetString(tmp.Balance, 10)
					if !ok {
						return fmt.Errorf("invalid balance string: %s", tmp.Balance)
					}
					msg = events.BalanceMessagePayload{
						AccountID:     tmp.AccountID,
						ConnectorID:   tmp.ConnectorID,
						Provider:      tmp.Provider,
						CreatedAt:     tmp.CreatedAt,
						LastUpdatedAt: tmp.LastUpdatedAt,
						Asset:         tmp.Asset,
						Balance:       bi,
					}
					return nil
				})))

			balanceResponse, err := app.GetValue().SDK().Payments.V1.GetAccountBalances(ctx, operations.GetAccountBalancesRequest{
				AccountID: msg.AccountID,
			})
			Expect(err).To(BeNil())
			res := balanceResponse.GetBalancesCursor()
			Expect(res.Cursor.Data).To(HaveLen(1))

			balance := res.Cursor.Data[0]
			Expect(balance.AccountID).To(Equal(msg.AccountID))
			Expect(balance.Balance).To(Equal(msg.Balance))
			Expect(balance.Asset).To(Equal(msg.Asset))
			Expect(balance.CreatedAt).To(Equal(msg.CreatedAt.UTC().Truncate(time.Second)))
		})

		It("should be successful with v3", func() {
			var msg events.BalanceMessagePayload
			Eventually(app.GetValue()).
				Should(OutboxEvent(models.OUTBOX_EVENT_BALANCE_SAVED, WithRawCallback(func(b []byte) error {
					var tmp events.BalanceMessagePayload
					if err := json.Unmarshal(b, &tmp); err != nil {
						return err
					}
					return nil
				})))

			balanceResponse, err := app.GetValue().SDK().Payments.V3.GetAccountBalances(ctx, operations.V3GetAccountBalancesRequest{
				AccountID: msg.AccountID,
			})
			Expect(err).To(BeNil())
			res := balanceResponse.GetV3BalancesCursorResponse()
			Expect(res.Cursor.Data).To(HaveLen(1))

			balance := res.Cursor.Data[0]
			Expect(balance.AccountID).To(Equal(msg.AccountID))
			Expect(balance.Balance).To(Equal(msg.Balance))
			Expect(balance.Asset).To(Equal(msg.Asset))
			Expect(balance.CreatedAt).To(Equal(msg.CreatedAt.UTC().Truncate(time.Second)))
		})
	})
})

// put near the test for debugging
//func dumpEvents(ch <-chan *nats.Msg, dur time.Duration) {
//	deadline := time.After(dur)
//	for {
//		select {
//		case <-deadline:
//			return
//		case m := <-ch:
//			if m == nil {
//				continue
//			}
//			type envelope struct {
//				Type    string          `json:"type"`
//				Payload json.RawMessage `json:"payload"`
//			}
//			var ev envelope
//			if err := json.Unmarshal(m.Data, &ev); err != nil {
//				GinkgoWriter.Printf("recv raw msg (unmarshal err: %v): %s\n", err, string(m.Data))
//				continue
//			}
//			GinkgoWriter.Printf("recv event: type=%s payload=%s\n", ev.Type, string(ev.Payload))
//		}
//	}
//}

func createV3Account(
	ctx context.Context,
	app *testserver.Server,
	req *components.V3CreateAccountRequest,
) (string, error) {
	createResponse, err := app.SDK().Payments.V3.CreateAccount(ctx, req)
	if err != nil {
		return "", err
	}
	return createResponse.GetV3CreateAccountResponse().Data.ID, nil
}

func createV2Account(
	ctx context.Context,
	app *testserver.Server,
	req components.AccountRequest,
) (string, error) {
	createResponse, err := app.SDK().Payments.V1.CreateAccount(ctx, req)
	if err != nil {
		return "", err
	}
	return createResponse.GetAccountResponse().Data.ID, nil
}
