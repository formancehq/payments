//go:build it

package test_suite

import (
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/testing/utils"
	v3 "github.com/formancehq/payments/internal/api/v3"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	evts "github.com/formancehq/payments/pkg/events"
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

		createRequest v3.CreateAccountRequest

		app *utils.Deferred[*Server]
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
	createRequest = v3.CreateAccountRequest{
		Reference:    "ref",
		AccountName:  "foo",
		CreatedAt:    createdAt,
		DefaultAsset: "USD",
		Type:         string(models.ACCOUNT_TYPE_INTERNAL),
		Metadata:     map[string]string{"key": "val"},
	}

	When("creating a new account", func() {
		var (
			connectorRes   struct{ Data string }
			createResponse struct{ Data models.Account }
			getResponse    struct{ Data models.Account }
			e              chan *nats.Msg
			err            error
		)

		DescribeTable("should be successful",
			func(ver int) {
				e = Subscribe(GinkgoT(), app.GetValue())
				connectorConf := newConnectorConfigurationFn()(uuid.New())
				err = ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
				Expect(err).To(BeNil())

				createRequest.ConnectorID = connectorRes.Data
				err = CreateAccount(ctx, app.GetValue(), ver, createRequest, &createResponse)
				Expect(err).To(BeNil())

				err = GetAccount(ctx, app.GetValue(), ver, createResponse.Data.ID.String(), &getResponse)
				Expect(err).To(BeNil())
				Expect(getResponse.Data).To(Equal(createResponse.Data))

				Eventually(e).Should(Receive(Event(evts.EventTypeSavedAccounts)))
			},
			Entry("with v2", 2),
			Entry("with v3", 3),
		)
	})

	When("fetching account balances", func() {
		var (
			connectorRes struct{ Data string }
			res          struct {
				Cursor bunpaginate.Cursor[models.Balance]
			}
			e chan *nats.Msg
		)

		DescribeTable("should be successful",
			func(ver int) {
				e = Subscribe(GinkgoT(), app.GetValue())
				connectorConf := newConnectorConfigurationFn()(uuid.New())
				_, err := GeneratePSPData(connectorConf.Directory)
				Expect(err).To(BeNil())
				ver = 3

				err = ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
				Expect(err).To(BeNil())
				Eventually(e).WithTimeout(2 * time.Second).Should(Receive(Event(evts.EventTypeSavedAccounts)))

				var msg events.BalanceMessagePayload
				// poll more frequently to filter out ACCOUNT_SAVED messages that we don't care about quicker
				Eventually(e).WithPolling(5 * time.Millisecond).WithTimeout(2 * time.Second).Should(Receive(Event(evts.EventTypeSavedBalances, WithCallback(
					msg,
					func(b []byte) error {
						return json.Unmarshal(b, &msg)
					},
				))))

				err = GetAccountBalances(ctx, app.GetValue(), ver, msg.AccountID, &res)
				Expect(err).To(BeNil())
				Expect(res.Cursor.Data).To(HaveLen(1))

				balance := res.Cursor.Data[0]
				Expect(balance.AccountID.String()).To(Equal(msg.AccountID))
				Expect(balance.Balance).To(Equal(msg.Balance))
				Expect(balance.Asset).To(Equal(msg.Asset))
				Expect(balance.CreatedAt).To(Equal(msg.CreatedAt.UTC().Truncate(time.Second)))
			},
			Entry("with v2", 2),
			Entry("with v3", 3),
		)
	})
})
