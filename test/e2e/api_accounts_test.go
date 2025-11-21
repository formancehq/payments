//go:build it

package test_suite

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	internalEvents "github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/pkg/client/models/components"
	"github.com/formancehq/payments/pkg/client/models/operations"
	"github.com/formancehq/payments/pkg/events"
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
			Stack:                      stack,
			PostgresConfiguration:      db.GetValue().ConnectionOptions(),
			NatsURL:                    natsServer.GetValue().ClientURL(),
			TemporalNamespace:          temporalServer.GetValue().DefaultNamespace(),
			TemporalAddress:            temporalServer.GetValue().Address(),
			Output:                     GinkgoWriter,
			SkipOutboxScheduleCreation: true,
		}
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

			n, err := CountOutboxEventsByType(ctx, app.GetValue(), events.EventTypeSavedAccounts)
			Expect(err).To(BeNil())
			Expect(n).To(Equal(1))
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

			n, err := CountOutboxEventsByType(ctx, app.GetValue(), events.EventTypeSavedAccounts)
			Expect(err).To(BeNil())
			Expect(n).To(Equal(1))
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
			_, err = GeneratePSPData(connectorConf.Directory, 5)
			Expect(err).To(BeNil())
			Eventually(func() int {
				n, _ := CountOutboxEventsByType(ctx, app.GetValue(), events.EventTypeSavedAccounts)
				return n
			}).WithTimeout(10 * time.Second).Should(Equal(5))
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should be successful with v2", func() {
			var msg internalEvents.BalanceMessagePayload
			Eventually(func() bool {
				payloads, err := LoadOutboxPayloadsByType(ctx, app.GetValue(), events.EventTypeSavedBalances)
				if err != nil {
					return false
				}
				for _, p := range payloads {
					var tmp internalEvents.BalanceMessagePayload
					if json.Unmarshal(p, &tmp) == nil {
						msg = tmp
						return true
					}
				}
				return false
			}).WithTimeout(10 * time.Second).Should(BeTrue())

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
			var msg internalEvents.BalanceMessagePayload
			Eventually(func() bool {
				payloads, err := LoadOutboxPayloadsByType(ctx, app.GetValue(), events.EventTypeSavedBalances)
				if err != nil {
					return false
				}
				for _, p := range payloads {
					var tmp internalEvents.BalanceMessagePayload
					if json.Unmarshal(p, &tmp) == nil {
						msg = tmp
						return true
					}
				}
				return false
			}).Should(BeTrue())

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
