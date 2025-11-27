//go:build it

package test_suite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	internalEvents "github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/client/models/components"
	"github.com/formancehq/payments/pkg/client/models/operations"
	"github.com/formancehq/payments/pkg/events"
	"github.com/formancehq/payments/pkg/testserver"
	"github.com/google/uuid"

	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Pools", Serial, func() {
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

	AfterEach(func() {
		flushRemainingWorkflows(ctx)
	})

	When("creating a new pool with v3", func() {
		var (
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

		It("should be ok when underlying accounts exist", func() {
			accountIDs := setupV3PoolAccounts(ctx, app.GetValue(), connectorID, 5)
			createResponse, err := app.GetValue().SDK().Payments.V3.CreatePool(ctx, &components.V3CreatePoolRequest{
				Name:       "some-pool",
				AccountIDs: accountIDs,
			})
			Expect(err).To(BeNil())

			poolID := createResponse.GetV3CreatePoolResponse().Data
			var msg = GenericEventPayload{ID: poolID}
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(msg))

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPool(ctx, poolID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetV3GetPoolResponse().Data.PoolAccounts).To(HaveLen(len(accountIDs)))
		})

		It("should be ok with a query", func() {
			accountIDs := setupV3PoolAccounts(ctx, app.GetValue(), connectorID, 5)
			createResponse, err := app.GetValue().SDK().Payments.V3.CreatePool(ctx, &components.V3CreatePoolRequest{
				Name: "some-pool",
				Query: map[string]any{
					"$match": map[string]any{
						"id": accountIDs[0],
					},
				},
			})
			Expect(err).To(BeNil())

			poolID := createResponse.GetV3CreatePoolResponse().Data
			var msg = GenericEventPayload{ID: poolID}
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(msg))

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPool(ctx, poolID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetV3GetPoolResponse().Data.PoolAccounts).To(HaveLen(1))
			Expect(getResponse.GetV3GetPoolResponse().Data.PoolAccounts[0]).To(Equal(accountIDs[0]))
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
			accountIDs := setupV3PoolAccounts(ctx, app.GetValue(), connectorID, 1)
			createResponse, err := app.GetValue().SDK().Payments.V3.CreatePool(ctx, &components.V3CreatePoolRequest{
				Name:       "some-pool",
				AccountIDs: accountIDs,
			})
			Expect(err).To(BeNil())

			poolID := createResponse.GetV3CreatePoolResponse().Data
			var msg = GenericEventPayload{ID: poolID}
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(msg))

			_, err = app.GetValue().SDK().Payments.V3.DeletePool(ctx, poolID)
			Expect(err).To(BeNil())
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeDeletePool, WithPayloadSubset(msg))
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
			err         error
		)

		BeforeEach(func() {
			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 2)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should be ok when underlying accounts exist", func() {
			accountIDs := setupV2PoolAccounts(ctx, app.GetValue(), connectorID, 5)
			createResponse, err := app.GetValue().SDK().Payments.V1.CreatePool(ctx, components.PoolRequest{
				Name:       "some-pool",
				AccountIDs: accountIDs,
			})
			Expect(err).To(BeNil())

			poolID := createResponse.GetPoolResponse().Data.ID
			var msg = GenericEventPayload{ID: poolID}
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(msg))

			getResponse, err := app.GetValue().SDK().Payments.V1.GetPool(ctx, poolID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetPoolResponse().Data.Accounts).To(HaveLen(len(accountIDs)))
		})

		It("should be ok with a query", func() {
			accountIDs := setupV2PoolAccounts(ctx, app.GetValue(), connectorID, 5)
			createResponse, err := app.GetValue().SDK().Payments.V1.CreatePool(ctx, components.PoolRequest{
				Name: "some-pool",
				Query: map[string]any{
					"$match": map[string]any{
						"id": accountIDs[0],
					},
				},
			})
			Expect(err).To(BeNil())

			poolID := createResponse.GetPoolResponse().Data.ID
			var msg = GenericEventPayload{ID: poolID}
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(msg))

			getResponse, err := app.GetValue().SDK().Payments.V1.GetPool(ctx, poolID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetPoolResponse().Data.Accounts).To(HaveLen(1))
			Expect(getResponse.GetPoolResponse().Data.Accounts[0]).To(Equal(accountIDs[0]))
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
			accountIDs := setupV2PoolAccounts(ctx, app.GetValue(), connectorID, 1)
			createResponse, err := app.GetValue().SDK().Payments.V1.CreatePool(ctx, components.PoolRequest{
				Name:       "some-pool",
				AccountIDs: accountIDs,
			})
			Expect(err).To(BeNil())

			poolID := createResponse.GetPoolResponse().Data.ID
			var msg = GenericEventPayload{ID: poolID}
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(msg))

			_, err = app.GetValue().SDK().Payments.V1.DeletePool(ctx, poolID)
			Expect(err).To(BeNil())
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeDeletePool, WithPayloadSubset(msg))
		})

		It("should not fail when attempting to delete a pool that doesn't exist", func() {
			poolID := uuid.New().String()
			_, err := app.GetValue().SDK().Payments.V1.DeletePool(ctx, poolID)
			Expect(err).To(BeNil())
		})
	})

	When("updating the query of a pool with v3", func() {
		var (
			connectorID     string
			accountIDs      []string
			err             error
			poolID          string
			poolWithQueryID string

			eventPayloadWithoutQuery GenericEventPayload
			eventPayloadWithQuery    GenericEventPayload
		)

		BeforeEach(func() {
			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 3)
			Expect(err).To(BeNil())
			ids := setupV3PoolAccounts(ctx, app.GetValue(), connectorID, 4)
			accountIDs = ids[0:2]

			createResponse, err := app.GetValue().SDK().Payments.V3.CreatePool(ctx, &components.V3CreatePoolRequest{
				Name:       "some-pool",
				AccountIDs: accountIDs,
			})
			Expect(err).To(BeNil())
			poolID = createResponse.GetV3CreatePoolResponse().Data
			eventPayloadWithoutQuery = GenericEventPayload{ID: poolID}
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(eventPayloadWithoutQuery))

			createResponse2, err := app.GetValue().SDK().Payments.V3.CreatePool(ctx, &components.V3CreatePoolRequest{
				Name: "some-pool-with query",
				Query: map[string]any{
					"$match": map[string]any{
						"id": accountIDs[0],
					},
				},
			})
			Expect(err).To(BeNil())
			poolWithQueryID = createResponse2.GetV3CreatePoolResponse().Data
			eventPayloadWithQuery = GenericEventPayload{ID: poolWithQueryID}
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(eventPayloadWithQuery))
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should not be possible to update the query of a static pool", func() {
			_, err := app.GetValue().SDK().Payments.V3.UpdatePoolQuery(ctx, poolID, &components.V3UpdatePoolQueryRequest{
				Query: map[string]any{
					"$match": map[string]any{
						"id": accountIDs[0],
					},
				},
			})
			Expect(err).To(Not(BeNil()))
		})

		It("should be possible to update the query of a dynamic pool", func() {
			_, err := app.GetValue().SDK().Payments.V3.UpdatePoolQuery(ctx, poolWithQueryID, &components.V3UpdatePoolQueryRequest{
				Query: map[string]any{
					"$match": map[string]any{
						"id": accountIDs[1],
					},
				},
			})
			Expect(err).To(BeNil())
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(eventPayloadWithQuery))

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPool(ctx, poolWithQueryID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetV3GetPoolResponse().Data.PoolAccounts).To(HaveLen(1))
			Expect(getResponse.GetV3GetPoolResponse().Data.PoolAccounts[0]).To(Equal(accountIDs[1]))
		})
	})

	When("updating the query of a pool with v2", func() {
		var (
			connectorID     string
			accountIDs      []string
			err             error
			poolID          string
			poolWithQueryID string

			eventPayloadWithoutQuery GenericEventPayload
			eventPayloadWithQuery    GenericEventPayload
		)

		BeforeEach(func() {
			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 2)
			Expect(err).To(BeNil())
			ids := setupV2PoolAccounts(ctx, app.GetValue(), connectorID, 4)
			accountIDs = ids[0:2]

			createResponse, err := app.GetValue().SDK().Payments.V1.CreatePool(ctx, components.PoolRequest{
				Name:       "some-pool",
				AccountIDs: accountIDs,
			})
			Expect(err).To(BeNil())
			poolID = createResponse.GetPoolResponse().Data.ID
			eventPayloadWithoutQuery = GenericEventPayload{ID: poolID}
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(eventPayloadWithoutQuery))

			createResponse2, err := app.GetValue().SDK().Payments.V1.CreatePool(ctx, components.PoolRequest{
				Name: "some-pool-with-query",
				Query: map[string]any{
					"$match": map[string]any{
						"id": accountIDs[0],
					},
				},
			})
			Expect(err).To(BeNil())
			poolWithQueryID = createResponse2.GetPoolResponse().Data.ID
			eventPayloadWithQuery = GenericEventPayload{ID: poolWithQueryID}
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(eventPayloadWithQuery))
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should not be possible to update the query of a static pool", func() {
			_, err := app.GetValue().SDK().Payments.V1.UpdatePoolQuery(ctx, poolID, components.UpdatePoolQueryRequest{
				Query: map[string]any{
					"$match": map[string]any{
						"id": accountIDs[0],
					},
				},
			})
			Expect(err).To(Not(BeNil()))
		})

		It("should be possible to update the query of a dynamic pool", func() {
			_, err := app.GetValue().SDK().Payments.V1.UpdatePoolQuery(ctx, poolWithQueryID, components.UpdatePoolQueryRequest{
				Query: map[string]any{
					"$match": map[string]any{
						"id": accountIDs[1],
					},
				},
			})
			Expect(err).To(BeNil())
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(eventPayloadWithQuery))

			getResponse, err := app.GetValue().SDK().Payments.V1.GetPool(ctx, poolWithQueryID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetPoolResponse().Data.Accounts).To(HaveLen(1))
			Expect(getResponse.GetPoolResponse().Data.Accounts[0]).To(Equal(accountIDs[1]))
		})
	})

	When("adding and removing accounts to a pool with v3", func() {
		var (
			connectorID     string
			accountIDs      []string
			extraAccountIDs []string
			poolID          string
			poolWithQueryID string
			err             error

			eventPayloadWithoutQuery GenericEventPayload
			eventPayloadWithQuery    GenericEventPayload
		)

		BeforeEach(func() {
			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 3)
			Expect(err).To(BeNil())
			ids := setupV3PoolAccounts(ctx, app.GetValue(), connectorID, 4)
			accountIDs = ids[0:2]
			extraAccountIDs = ids[2:4]

			createResponse, err := app.GetValue().SDK().Payments.V3.CreatePool(ctx, &components.V3CreatePoolRequest{
				Name:       "some-pool",
				AccountIDs: accountIDs,
			})
			Expect(err).To(BeNil())
			poolID = createResponse.GetV3CreatePoolResponse().Data
			eventPayloadWithoutQuery = GenericEventPayload{ID: poolID}
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(eventPayloadWithoutQuery))
			createResponse2, err := app.GetValue().SDK().Payments.V3.CreatePool(ctx, &components.V3CreatePoolRequest{
				Name: "some-pool-with query",
				Query: map[string]any{
					"$match": map[string]any{
						"id": accountIDs[0],
					},
				},
			})
			Expect(err).To(BeNil())
			poolWithQueryID = createResponse2.GetV3CreatePoolResponse().Data
			eventPayloadWithQuery = GenericEventPayload{ID: poolWithQueryID}
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(eventPayloadWithQuery))
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should be possible to remove account from static pool", func() {
			_, err := app.GetValue().SDK().Payments.V3.RemoveAccountFromPool(ctx, poolID, accountIDs[0])
			Expect(err).To(BeNil())
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(eventPayloadWithoutQuery))

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPool(ctx, poolID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetV3GetPoolResponse().Data.PoolAccounts).To(HaveLen(len(accountIDs) - 1))
			Expect(getResponse.GetV3GetPoolResponse().Data.PoolAccounts[0]).To(Equal(accountIDs[1]))
		})

		It("should not be possible to remove account from dynamic pool", func() {
			_, err := app.GetValue().SDK().Payments.V3.RemoveAccountFromPool(ctx, poolWithQueryID, accountIDs[0])
			Expect(err).To(Not(BeNil()))

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPool(ctx, poolWithQueryID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetV3GetPoolResponse().Data.PoolAccounts).To(HaveLen(1))
			Expect(getResponse.GetV3GetPoolResponse().Data.PoolAccounts[0]).To(Equal(accountIDs[0]))
		})

		It("should not fail even when removing underlying account not attached to pool", func() {
			_, err := app.GetValue().SDK().Payments.V3.RemoveAccountFromPool(ctx, poolID, extraAccountIDs[0])
			Expect(err).To(BeNil())
		})

		It("should be possible to add account to static pool", func() {
			_, err := app.GetValue().SDK().Payments.V3.AddAccountToPool(ctx, poolID, extraAccountIDs[0])
			Expect(err).To(BeNil())
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(eventPayloadWithoutQuery))

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPool(ctx, poolID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetV3GetPoolResponse().Data.PoolAccounts).To(HaveLen(len(accountIDs) + 1))
		})

		It("should not be possible to add account to dynamic pool", func() {
			_, err := app.GetValue().SDK().Payments.V3.AddAccountToPool(ctx, poolWithQueryID, extraAccountIDs[0])
			Expect(err).To(Not(BeNil()))

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPool(ctx, poolWithQueryID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetV3GetPoolResponse().Data.PoolAccounts).To(HaveLen(1))
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
			poolWithQueryID string
			err             error

			eventPayloadWithoutQuery GenericEventPayload
			eventPayloadWithQuery    GenericEventPayload
		)

		BeforeEach(func() {
			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 2)
			Expect(err).To(BeNil())
			ids := setupV2PoolAccounts(ctx, app.GetValue(), connectorID, 4)
			accountIDs = ids[0:2]
			extraAccountIDs = ids[2:4]

			createResponse, err := app.GetValue().SDK().Payments.V1.CreatePool(ctx, components.PoolRequest{
				Name:       "some-pool",
				AccountIDs: accountIDs,
			})
			Expect(err).To(BeNil())
			poolID = createResponse.GetPoolResponse().Data.ID
			eventPayloadWithoutQuery = GenericEventPayload{ID: poolID}
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(eventPayloadWithoutQuery))

			createResponse2, err := app.GetValue().SDK().Payments.V1.CreatePool(ctx, components.PoolRequest{
				Name: "some-pool-with-query",
				Query: map[string]any{
					"$match": map[string]any{
						"id": accountIDs[0],
					},
				},
			})
			Expect(err).To(BeNil())
			poolWithQueryID = createResponse2.GetPoolResponse().Data.ID
			eventPayloadWithQuery = GenericEventPayload{ID: poolWithQueryID}
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(eventPayloadWithQuery))
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should be possible to remove account from static pool", func() {
			_, err := app.GetValue().SDK().Payments.V1.RemoveAccountFromPool(ctx, poolID, accountIDs[0])
			Expect(err).To(BeNil())
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(eventPayloadWithoutQuery))

			getResponse, err := app.GetValue().SDK().Payments.V1.GetPool(ctx, poolID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetPoolResponse().Data.Accounts).To(HaveLen(len(accountIDs) - 1))
			Expect(getResponse.GetPoolResponse().Data.Accounts[0]).To(Equal(accountIDs[1]))
		})

		It("should not be possible to remove account from dynamic pool", func() {
			_, err := app.GetValue().SDK().Payments.V1.RemoveAccountFromPool(ctx, poolWithQueryID, accountIDs[0])
			Expect(err).To(Not(BeNil()))

			getResponse, err := app.GetValue().SDK().Payments.V1.GetPool(ctx, poolWithQueryID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetPoolResponse().Data.Accounts).To(HaveLen(1))
			Expect(getResponse.GetPoolResponse().Data.Accounts[0]).To(Equal(accountIDs[0]))
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
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(eventPayloadWithoutQuery))

			getResponse, err := app.GetValue().SDK().Payments.V1.GetPool(ctx, poolID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetPoolResponse().Data.Accounts).To(HaveLen(len(accountIDs) + 1))
		})

		It("should not be possible to add account to pool", func() {
			_, err := app.GetValue().SDK().Payments.V1.AddAccountToPool(ctx, poolWithQueryID, components.AddAccountToPoolRequest{
				AccountID: extraAccountIDs[0],
			})
			Expect(err).To(Not(BeNil()))

			getResponse, err := app.GetValue().SDK().Payments.V1.GetPool(ctx, poolWithQueryID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetPoolResponse().Data.Accounts).To(HaveLen(1))
		})

		It("should not fail event when adding underlying account already in pool", func() {
			_, err := app.GetValue().SDK().Payments.V1.AddAccountToPool(ctx, poolID, components.AddAccountToPoolRequest{
				AccountID: accountIDs[0],
			})
			Expect(err).To(BeNil())
		})
	})

	When("fetching balances for a pool", func() {
		var (
			connectorID     string
			accountIDs      []string
			balance         components.AccountBalance
			poolID          string
			poolWithQueryID string
			err             error

			eventPayloadWithQuery    GenericEventPayload
			eventPayloadWithoutQuery GenericEventPayload
		)

		BeforeEach(func() {
			id := uuid.New()
			connectorConf := newV3ConnectorConfigFn()(id)

			connectorID, err = installV3Connector(ctx, app.GetValue(), connectorConf, uuid.New())
			Expect(err).To(BeNil())

			_, err = GeneratePSPData(connectorConf.Directory, 1)
			Expect(err).To(BeNil())

			// Wait for at least one account to be created to ensure the connector workflow has started
			Eventually(func() int {
				n, _ := CountOutboxEventsByType(ctx, app.GetValue(), events.EventTypeSavedAccounts)
				return n
			}).WithTimeout(10 * time.Second).WithPolling(100 * time.Millisecond).Should(Equal(1))

			// Wait for balance event to appear in the outbox
			var msg internalEvents.BalanceMessagePayload
			Eventually(func() bool {
				payloads, err := LoadOutboxPayloadsByType(ctx, app.GetValue(), events.EventTypeSavedBalances)
				if err != nil {
					return false
				}
				for _, p := range payloads {
					var tmp internalEvents.BalanceMessagePayload
					if json.Unmarshal(p, &tmp) == nil && tmp.AccountID != "" {
						msg = tmp
						return true
					}
				}
				return false
			}).WithTimeout(10 * time.Second).WithPolling(100 * time.Millisecond).Should(BeTrue())

			balanceResponse, err := app.GetValue().SDK().Payments.V1.GetAccountBalances(ctx, operations.GetAccountBalancesRequest{
				AccountID: msg.AccountID,
			})
			Expect(err).To(BeNil())
			res := balanceResponse.GetBalancesCursor()
			Expect(res.Cursor.Data).To(HaveLen(1))

			balance = res.Cursor.Data[0]
			accountIDs = []string{balance.AccountID}

			createResponse, err := app.GetValue().SDK().Payments.V3.CreatePool(ctx, &components.V3CreatePoolRequest{
				Name:       "some-pool",
				AccountIDs: accountIDs,
			})
			Expect(err).To(BeNil())
			poolID = createResponse.GetV3CreatePoolResponse().Data
			eventPayloadWithoutQuery = GenericEventPayload{ID: poolID}
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(eventPayloadWithoutQuery))

			createResponse2, err := app.GetValue().SDK().Payments.V3.CreatePool(ctx, &components.V3CreatePoolRequest{
				Name: "some-pool-with-query",
				Query: map[string]any{
					"$match": map[string]any{
						"id": balance.AccountID,
					},
				},
			})
			Expect(err).To(BeNil())
			poolWithQueryID = createResponse2.GetV3CreatePoolResponse().Data
			eventPayloadWithQuery = GenericEventPayload{ID: poolWithQueryID}
			MustEventuallyOutbox(ctx, app.GetValue(), events.EventTypeSavedPool, WithPayloadSubset(eventPayloadWithQuery))
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should fetch balances for accounts in static pool using v3", func() {
			res, err := app.GetValue().SDK().Payments.V3.GetPoolBalancesLatest(ctx, poolID)
			Expect(err).To(BeNil())
			Expect(res.GetV3PoolBalancesResponse().Data).To(HaveLen(1))
			Expect(res.GetV3PoolBalancesResponse().Data[0].GetRelatedAccounts()).To(Equal([]string{balance.GetAccountID()}))
			Expect(res.GetV3PoolBalancesResponse().Data[0].GetAsset()).To(Equal(balance.GetAsset()))
			Expect(res.GetV3PoolBalancesResponse().Data[0].GetAmount()).To(Equal(balance.GetBalance()))
		})

		It("should fetch balances for accounts in dynamic pool using v3", func() {
			res, err := app.GetValue().SDK().Payments.V3.GetPoolBalancesLatest(ctx, poolWithQueryID)
			Expect(err).To(BeNil())
			Expect(res.GetV3PoolBalancesResponse().Data).To(HaveLen(1))
			Expect(res.GetV3PoolBalancesResponse().Data[0].GetRelatedAccounts()).To(Equal([]string{balance.GetAccountID()}))
			Expect(res.GetV3PoolBalancesResponse().Data[0].GetAsset()).To(Equal(balance.GetAsset()))
			Expect(res.GetV3PoolBalancesResponse().Data[0].GetAmount()).To(Equal(balance.GetBalance()))
		})

		It("should fetch balances for accounts in static pool using v2", func() {
			res, err := app.GetValue().SDK().Payments.V1.GetPoolBalancesLatest(ctx, poolID)
			Expect(err).To(BeNil())
			Expect(res.GetPoolBalancesLatestResponse().Data).To(HaveLen(1))
			Expect(res.GetPoolBalancesLatestResponse().Data[0].GetRelatedAccounts()).To(Equal([]string{balance.GetAccountID()}))
			Expect(res.GetPoolBalancesLatestResponse().Data[0].GetAsset()).To(Equal(balance.GetAsset()))
			Expect(res.GetPoolBalancesLatestResponse().Data[0].GetAmount()).To(Equal(balance.GetBalance()))
		})

		It("should fetch balances for accounts in dynamic pool using v2", func() {
			res, err := app.GetValue().SDK().Payments.V1.GetPoolBalancesLatest(ctx, poolWithQueryID)
			Expect(err).To(BeNil())
			Expect(res.GetPoolBalancesLatestResponse().Data).To(HaveLen(1))
			Expect(res.GetPoolBalancesLatestResponse().Data[0].GetRelatedAccounts()).To(Equal([]string{balance.GetAccountID()}))
			Expect(res.GetPoolBalancesLatestResponse().Data[0].GetAsset()).To(Equal(balance.GetAsset()))
			Expect(res.GetPoolBalancesLatestResponse().Data[0].GetAmount()).To(Equal(balance.GetBalance()))
		})
	})
})

func setupV3PoolAccounts(
	ctx context.Context,
	app *testserver.Server,
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

		MustOutbox(ctx, app, events.EventTypeSavedAccounts, WithPayloadSubset(msg))
		accountIDs = append(accountIDs, accountID)
	}
	return accountIDs
}

func setupV2PoolAccounts(
	ctx context.Context,
	app *testserver.Server,
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
		MustEventuallyOutbox(ctx, app, events.EventTypeSavedAccounts, WithPayloadSubset(msg))
		accountIDs = append(accountIDs, accountID)
	}
	return accountIDs
}
