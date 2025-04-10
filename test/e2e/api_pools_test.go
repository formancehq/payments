package test_suite

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/client/models/components"
	evts "github.com/formancehq/payments/pkg/events"
	"github.com/formancehq/payments/pkg/testserver"
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

	When("creating a new pool with v3", func() {
		var (
			connectorID string
			e           chan *nats.Msg
			err         error
		)

		BeforeEach(func() {
			e = Subscribe(GinkgoT(), app.GetValue())
			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 3)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should be ok when underlying accounts exist", func() {
			accountIDs := setupV3PoolAccounts(ctx, app.GetValue(), e, connectorID, 5)
			createResponse, err := app.GetValue().SDK().Payments.V3.CreatePool(ctx, &components.V3CreatePoolRequest{
				Name:       "some-pool",
				AccountIDs: accountIDs,
			})
			Expect(err).To(BeNil())

			poolID := createResponse.GetV3CreatePoolResponse().Data
			var msg = GenericEventPayload{ID: poolID}
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPool, WithPayloadSubset(msg))))

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPool(ctx, poolID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetV3GetPoolResponse().Data.PoolAccounts).To(HaveLen(len(accountIDs)))
		})

		It("should fail when underlying accounts don't exist", func() {
			accountID := models.AccountID{
				Reference:   "v3blahblahblah",
				ConnectorID: models.MustConnectorIDFromString(connectorID),
			}
			_, err := app.GetValue().SDK().Payments.V3.CreatePool(ctx, &components.V3CreatePoolRequest{
				Name:       "some-pool",
				AccountIDs: []string{accountID.String()},
			})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("404"))
		})

		It("should be possible to delete a pool", func() {
			accountIDs := setupV3PoolAccounts(ctx, app.GetValue(), e, connectorID, 1)
			createResponse, err := app.GetValue().SDK().Payments.V3.CreatePool(ctx, &components.V3CreatePoolRequest{
				Name:       "some-pool",
				AccountIDs: accountIDs,
			})
			Expect(err).To(BeNil())

			poolID := createResponse.GetV3CreatePoolResponse().Data
			var msg = GenericEventPayload{ID: poolID}
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPool, WithPayloadSubset(msg))))

			_, err = app.GetValue().SDK().Payments.V3.DeletePool(ctx, poolID)
			Expect(err).To(BeNil())
			Eventually(e).Should(Receive(Event(evts.EventTypeDeletePool, WithPayloadSubset(msg))))
		})

		It("should not fail when attempting to delete a pool that doesn't exist", func() {
			poolID := uuid.New().String()
			_, err := app.GetValue().SDK().Payments.V3.DeletePool(ctx, poolID)
			Expect(err).To(BeNil())
		})
	})

	When("creating a new pool with v2", func() {
		var (
			connectorID string
			e           chan *nats.Msg
			err         error
		)

		BeforeEach(func() {
			e = Subscribe(GinkgoT(), app.GetValue())
			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 2)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should be ok when underlying accounts exist", func() {
			accountIDs := setupV2PoolAccounts(ctx, app.GetValue(), e, connectorID, 5)
			createResponse, err := app.GetValue().SDK().Payments.V1.CreatePool(ctx, components.PoolRequest{
				Name:       "some-pool",
				AccountIDs: accountIDs,
			})
			Expect(err).To(BeNil())

			poolID := createResponse.GetPoolResponse().Data.ID
			var msg = GenericEventPayload{ID: poolID}
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPool, WithPayloadSubset(msg))))

			getResponse, err := app.GetValue().SDK().Payments.V1.GetPool(ctx, poolID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetPoolResponse().Data.Accounts).To(HaveLen(len(accountIDs)))
		})

		It("should fail when underlying accounts don't exist", func() {
			accountID := models.AccountID{
				Reference:   "blahblahblah",
				ConnectorID: models.MustConnectorIDFromString(connectorID),
			}
			_, err := app.GetValue().SDK().Payments.V1.CreatePool(ctx, components.PoolRequest{
				Name:       "blahblahblah",
				AccountIDs: []string{accountID.String()},
			})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("404"))
		})

		It("should be possible to delete a pool", func() {
			accountIDs := setupV2PoolAccounts(ctx, app.GetValue(), e, connectorID, 1)
			createResponse, err := app.GetValue().SDK().Payments.V1.CreatePool(ctx, components.PoolRequest{
				Name:       "some-pool",
				AccountIDs: accountIDs,
			})
			Expect(err).To(BeNil())

			poolID := createResponse.GetPoolResponse().Data.ID
			var msg = GenericEventPayload{ID: poolID}
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPool, WithPayloadSubset(msg))))

			_, err = app.GetValue().SDK().Payments.V1.DeletePool(ctx, poolID)
			Expect(err).To(BeNil())
			Eventually(e).Should(Receive(Event(evts.EventTypeDeletePool, WithPayloadSubset(msg))))
		})

		It("should not fail when attempting to delete a pool that doesn't exist", func() {
			poolID := uuid.New().String()
			_, err := app.GetValue().SDK().Payments.V1.DeletePool(ctx, poolID)
			Expect(err).To(BeNil())
		})
	})

	When("adding and removing accounts to a pool with v3", func() {
		var (
			connectorID     string
			accountIDs      []string
			extraAccountIDs []string
			poolID          string
			e               chan *nats.Msg
			err             error

			eventPayload GenericEventPayload
		)

		BeforeEach(func() {
			e = Subscribe(GinkgoT(), app.GetValue())
			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 3)
			Expect(err).To(BeNil())
			ids := setupV3PoolAccounts(ctx, app.GetValue(), e, connectorID, 4)
			accountIDs = ids[0:2]
			extraAccountIDs = ids[2:4]

			createResponse, err := app.GetValue().SDK().Payments.V3.CreatePool(ctx, &components.V3CreatePoolRequest{
				Name:       "some-pool",
				AccountIDs: accountIDs,
			})
			Expect(err).To(BeNil())
			poolID = createResponse.GetV3CreatePoolResponse().Data
			eventPayload = GenericEventPayload{ID: poolID}
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPool, WithPayloadSubset(eventPayload))))
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should be possible to remove account from pool", func() {
			_, err := app.GetValue().SDK().Payments.V3.RemoveAccountFromPool(ctx, poolID, accountIDs[0])
			Expect(err).To(BeNil())
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPool, WithPayloadSubset(eventPayload))))

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPool(ctx, poolID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetV3GetPoolResponse().Data.PoolAccounts).To(HaveLen(len(accountIDs) - 1))
			Expect(getResponse.GetV3GetPoolResponse().Data.PoolAccounts[0]).To(Equal(accountIDs[1]))
		})

		It("should not fail even when removing underlying account not attached to pool", func() {
			_, err := app.GetValue().SDK().Payments.V3.RemoveAccountFromPool(ctx, poolID, extraAccountIDs[0])
			Expect(err).To(BeNil())
		})

		It("should be possible to add account to pool", func() {
			_, err := app.GetValue().SDK().Payments.V3.AddAccountToPool(ctx, poolID, extraAccountIDs[0])
			Expect(err).To(BeNil())
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPool, WithPayloadSubset(eventPayload))))

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPool(ctx, poolID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetV3GetPoolResponse().Data.PoolAccounts).To(HaveLen(len(accountIDs) + 1))
		})

		It("should not fail event when adding underlying account already in pool", func() {
			_, err := app.GetValue().SDK().Payments.V3.AddAccountToPool(ctx, poolID, accountIDs[0])
			Expect(err).To(BeNil())
		})
	})

	When("adding and removing accounts to a pool with v2", func() {
		var (
			connectorID     string
			accountIDs      []string
			extraAccountIDs []string
			poolID          string
			e               chan *nats.Msg
			err             error

			eventPayload GenericEventPayload
		)

		BeforeEach(func() {
			e = Subscribe(GinkgoT(), app.GetValue())
			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 2)
			Expect(err).To(BeNil())
			ids := setupV2PoolAccounts(ctx, app.GetValue(), e, connectorID, 4)
			accountIDs = ids[0:2]
			extraAccountIDs = ids[2:4]

			createResponse, err := app.GetValue().SDK().Payments.V1.CreatePool(ctx, components.PoolRequest{
				Name:       "some-pool",
				AccountIDs: accountIDs,
			})
			Expect(err).To(BeNil())
			poolID = createResponse.GetPoolResponse().Data.ID
			eventPayload = GenericEventPayload{ID: poolID}
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPool, WithPayloadSubset(eventPayload))))
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should be possible to remove account from pool", func() {
			_, err := app.GetValue().SDK().Payments.V1.RemoveAccountFromPool(ctx, poolID, accountIDs[0])
			Expect(err).To(BeNil())
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPool, WithPayloadSubset(eventPayload))))

			getResponse, err := app.GetValue().SDK().Payments.V1.GetPool(ctx, poolID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetPoolResponse().Data.Accounts).To(HaveLen(len(accountIDs) - 1))
			Expect(getResponse.GetPoolResponse().Data.Accounts[0]).To(Equal(accountIDs[1]))
		})

		It("should not fail even when removing underlying account not attached to pool", func() {
			_, err := app.GetValue().SDK().Payments.V1.RemoveAccountFromPool(ctx, poolID, extraAccountIDs[0])
			Expect(err).To(BeNil())
		})

		It("should be possible to add account to pool", func() {
			_, err := app.GetValue().SDK().Payments.V1.AddAccountToPool(ctx, poolID, components.AddAccountToPoolRequest{
				AccountID: extraAccountIDs[0],
			})
			Expect(err).To(BeNil())
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPool, WithPayloadSubset(eventPayload))))

			getResponse, err := app.GetValue().SDK().Payments.V1.GetPool(ctx, poolID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetPoolResponse().Data.Accounts).To(HaveLen(len(accountIDs) + 1))
		})

		It("should not fail event when adding underlying account already in pool", func() {
			_, err := app.GetValue().SDK().Payments.V1.AddAccountToPool(ctx, poolID, components.AddAccountToPoolRequest{
				AccountID: accountIDs[0],
			})
			Expect(err).To(BeNil())
		})
	})
})

func setupV3PoolAccounts(
	ctx context.Context,
	app *testserver.Server,
	e chan *nats.Msg,
	connectorID string,
	count int,
) []string {
	accountIDs := make([]string, 0, count)
	for i := 0; i < count; i++ {
		reference := fmt.Sprintf("account%d-ref", i)
		accountID, err := createV3Account(ctx, app, &components.V3CreateAccountRequest{
			Reference:   reference,
			ConnectorID: connectorID,
			CreatedAt:   time.Now().Truncate(time.Second),
			AccountName: fmt.Sprintf("account%d-name", i),
			Type:        "INTERNAL",
			Metadata:    map[string]string{"key": "val"},
		})
		Expect(err).To(BeNil())
		var msg = struct {
			ConnectorID string `json:"connectorID"`
			AccountID   string `json:"id"`
			Reference   string `json:"reference"`
		}{
			ConnectorID: connectorID,
			AccountID:   accountID,
			Reference:   reference,
		}
		Eventually(e).Should(Receive(Event(evts.EventTypeSavedAccounts, WithPayloadSubset(msg))))
		accountIDs = append(accountIDs, accountID)
	}
	return accountIDs
}

func setupV2PoolAccounts(
	ctx context.Context,
	app *testserver.Server,
	e chan *nats.Msg,
	connectorID string,
	count int,
) []string {
	accountIDs := make([]string, 0, count)
	for i := 0; i < count; i++ {
		reference := fmt.Sprintf("account%d-ref", i)
		accountID, err := createV2Account(ctx, app, components.AccountRequest{
			Reference:   reference,
			ConnectorID: connectorID,
			CreatedAt:   time.Now().Truncate(time.Second),
			AccountName: pointer.For(fmt.Sprintf("account%d-name", i)),
			Type:        "INTERNAL",
			Metadata:    map[string]string{"key": "val"},
		})
		Expect(err).To(BeNil())
		var msg = struct {
			ConnectorID string `json:"connectorID"`
			AccountID   string `json:"id"`
			Reference   string `json:"reference"`
		}{
			ConnectorID: connectorID,
			AccountID:   accountID,
			Reference:   reference,
		}
		Eventually(e).Should(Receive(Event(evts.EventTypeSavedAccounts, WithPayloadSubset(msg))))
		accountIDs = append(accountIDs, accountID)
	}
	return accountIDs
}
