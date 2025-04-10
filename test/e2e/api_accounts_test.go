//go:build it

package test_suite

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/pkg/client/models/components"
	"github.com/formancehq/payments/pkg/client/models/operations"
	evts "github.com/formancehq/payments/pkg/events"
	"github.com/formancehq/payments/pkg/testserver"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Accounts", func() {
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

	createdAt, _ := time.Parse("2006-Jan-02", "2024-Nov-29")

	When("creating a new account", func() {
		var (
			e           chan *nats.Msg
			connectorID string
			err         error
		)

		BeforeEach(func() {
			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 3)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should be successful with v2", func() {
			e = Subscribe(GinkgoT(), app.GetValue())
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

			Eventually(e).Should(Receive(Event(evts.EventTypeSavedAccounts)))
		})

		It("should be successful with v3", func() {
			e = Subscribe(GinkgoT(), app.GetValue())
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

			Eventually(e).Should(Receive(Event(evts.EventTypeSavedAccounts)))
		})
	})

	When("fetching account balances", func() {
		var (
			e           chan *nats.Msg
			connectorID string
			err         error
		)

		BeforeEach(func() {
			e = Subscribe(GinkgoT(), app.GetValue())
			id := uuid.New()
			connectorConf := newV3ConnectorConfigFn()(id)
			connectorID, err = installV3Connector(ctx, app.GetValue(), connectorConf, uuid.New())
			Expect(err).To(BeNil())
			_, err = GeneratePSPData(connectorConf.Directory)
			Expect(err).To(BeNil())
			Eventually(e).WithTimeout(2 * time.Second).Should(Receive(Event(evts.EventTypeSavedAccounts)))
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should be successful with v2", func() {
			var msg events.BalanceMessagePayload
			// poll more frequently to filter out ACCOUNT_SAVED messages that we don't care about quicker
			Eventually(e).WithTimeout(2 * time.Second).WithPolling(5 * time.Millisecond).Should(Receive(Event(evts.EventTypeSavedBalances, WithCallback(
				msg,
				func(b []byte) error {
					return json.Unmarshal(b, &msg)
				},
			))))

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
			// poll more frequently to filter out ACCOUNT_SAVED messages that we don't care about quicker
			Eventually(e).WithTimeout(2 * time.Second).WithPolling(5 * time.Millisecond).Should(Receive(Event(evts.EventTypeSavedBalances, WithCallback(
				msg,
				func(b []byte) error {
					return json.Unmarshal(b, &msg)
				},
			))))

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
