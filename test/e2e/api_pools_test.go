package test_suite

import (
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/testing/utils"
	v2 "github.com/formancehq/payments/internal/api/v2"
	v3 "github.com/formancehq/payments/internal/api/v3"
	"github.com/formancehq/payments/internal/models"
	evts "github.com/formancehq/payments/pkg/events"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Pools", func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()

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

	When("creating a new pool with v3", func() {
		var (
			connectorRes struct{ Data string }
			connectorID  string
			accountIDs   []string
			e            chan *nats.Msg
			ver          int
			err          error
		)

		JustBeforeEach(func() {
			ver = 3
			e = Subscribe(GinkgoT(), app.GetValue())
			connectorConf := newConnectorConfigurationFn()(uuid.New())
			err = ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())
			connectorID = connectorRes.Data

			for i := 0; i < 5; i++ {
				var accountResponse struct{ Data models.Account }
				accountRequest := v3.CreateAccountRequest{
					Reference:   fmt.Sprintf("account%d-ref", i),
					AccountName: fmt.Sprintf("account%d-name", i),
					ConnectorID: connectorID,
					CreatedAt:   time.Now().Truncate(time.Second),
					Type:        string(models.ACCOUNT_TYPE_INTERNAL),
					Metadata:    map[string]string{"key": "val"},
				}

				err = CreateAccount(ctx, app.GetValue(), ver, accountRequest, &accountResponse)
				Expect(err).To(BeNil())
				var msg = struct {
					ConnectorID string `json:"connectorId"`
					AccountID   string `json:"id"`
					Reference   string `json:"reference"`
				}{
					ConnectorID: connectorID,
					AccountID:   accountResponse.Data.ID.String(),
					Reference:   accountRequest.Reference,
				}
				Eventually(e).Should(Receive(Event(evts.EventTypeSavedAccounts, WithPayloadSubset(msg))))
				accountIDs = append(accountIDs, accountResponse.Data.ID.String())
			}
		})

		It("should be ok when underlying accounts exist", func() {
			req := v3.CreatePoolRequest{
				Name:       "some-pool",
				AccountIDs: accountIDs,
			}
			var res struct{ Data string }
			err = CreatePool(ctx, app.GetValue(), ver, req, &res)
			Expect(err).To(BeNil())

			poolID := res.Data
			var msg2 = struct {
				ID string `json:"id"`
			}{
				ID: poolID,
			}
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPool, WithPayloadSubset(msg2))))

			var getRes struct{ Data models.Pool }
			err = GetPool(ctx, app.GetValue(), ver, poolID, &getRes)
			Expect(err).To(BeNil())
			Expect(getRes.Data.PoolAccounts).To(HaveLen(len(accountIDs)))
		})

		It("should fail when underlying accounts don't exist", func() {
			accountID := models.AccountID{
				Reference:   "v3blahblahblah",
				ConnectorID: models.MustConnectorIDFromString(connectorID),
			}
			req := v3.CreatePoolRequest{
				Name:       "some-pool",
				AccountIDs: []string{accountID.String()},
			}
			var res struct{ Data v2.PoolResponse }
			err = CreatePool(ctx, app.GetValue(), ver, req, &res)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("404"))
		})
	})

	When("creating a new pool with v2", func() {
		var (
			connectorRes struct{ Data string }
			connectorID  string
			accountIDs   []string
			e            chan *nats.Msg
			ver          int
			err          error
		)

		JustBeforeEach(func() {
			ver = 2
			e = Subscribe(GinkgoT(), app.GetValue())
			connectorConf := newConnectorConfigurationFn()(uuid.New())
			err = ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())
			connectorID = connectorRes.Data

			for i := 0; i < 5; i++ {
				var accountResponse struct{ Data models.Account }
				accountRequest := v3.CreateAccountRequest{
					Reference:   fmt.Sprintf("account%d-ref", i),
					AccountName: fmt.Sprintf("account%d-name", i),
					ConnectorID: connectorID,
					CreatedAt:   time.Now().Truncate(time.Second),
					Type:        string(models.ACCOUNT_TYPE_INTERNAL),
					Metadata:    map[string]string{"key": "val"},
				}

				err = CreateAccount(ctx, app.GetValue(), ver, accountRequest, &accountResponse)
				Expect(err).To(BeNil())
				var msg = struct {
					ConnectorID string `json:"connectorId"`
					AccountID   string `json:"id"`
					Reference   string `json:"reference"`
				}{
					ConnectorID: connectorID,
					AccountID:   accountResponse.Data.ID.String(),
					Reference:   accountRequest.Reference,
				}
				Eventually(e).Should(Receive(Event(evts.EventTypeSavedAccounts, WithPayloadSubset(msg))))
				accountIDs = append(accountIDs, accountResponse.Data.ID.String())
			}
		})

		It("should be ok when underlying accounts exist", func() {
			req := v2.CreatePoolRequest{
				Name:       "some-pool",
				AccountIDs: accountIDs,
			}
			var res struct{ Data v2.PoolResponse }
			err = CreatePool(ctx, app.GetValue(), ver, req, &res)
			Expect(err).To(BeNil())

			poolID := res.Data.ID
			var msg2 = struct {
				ID string `json:"id"`
			}{
				ID: poolID,
			}
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPool, WithPayloadSubset(msg2))))

			var getRes struct{ Data v2.PoolResponse }
			err = GetPool(ctx, app.GetValue(), ver, poolID, &getRes)
			Expect(err).To(BeNil())
			Expect(getRes.Data.Accounts).To(HaveLen(len(accountIDs)))
		})

		It("should fail when underlying accounts don't exist", func() {
			accountID := models.AccountID{
				Reference:   "blahblahblah",
				ConnectorID: models.MustConnectorIDFromString(connectorID),
			}
			req := v2.CreatePoolRequest{
				Name:       "some-pool",
				AccountIDs: []string{accountID.String()},
			}
			var res struct{ Data v2.PoolResponse }
			err = CreatePool(ctx, app.GetValue(), ver, req, &res)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("404"))
		})
	})
})
