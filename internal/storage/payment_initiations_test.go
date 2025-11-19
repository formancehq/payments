package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	piID1 = models.PaymentInitiationID{
		Reference:   "test1",
		ConnectorID: defaultConnector.ID,
	}

	piID2 = models.PaymentInitiationID{
		Reference:   "test2",
		ConnectorID: defaultConnector.ID,
	}

	piID3 = models.PaymentInitiationID{
		Reference:   "test3",
		ConnectorID: defaultConnector.ID,
	}
)

func defaultPaymentInitiations() []models.PaymentInitiation {
	defaultAccounts := defaultAccounts()
	return []models.PaymentInitiation{
		{
			ID:                   piID1,
			ConnectorID:          defaultConnector.ID,
			Reference:            "test1",
			CreatedAt:            now.Add(-60 * time.Minute).UTC().Time,
			ScheduledAt:          now.Add(-60 * time.Minute).UTC().Time,
			Description:          "test1",
			Type:                 models.PAYMENT_INITIATION_TYPE_PAYOUT,
			DestinationAccountID: &defaultAccounts[0].ID,
			Amount:               big.NewInt(100),
			Asset:                "EUR/2",
		},
		{
			ID:                   piID2,
			ConnectorID:          defaultConnector.ID,
			Reference:            "test2",
			CreatedAt:            now.Add(-30 * time.Minute).UTC().Time,
			ScheduledAt:          now.Add(-20 * time.Minute).UTC().Time,
			Description:          "test2",
			Type:                 models.PAYMENT_INITIATION_TYPE_TRANSFER,
			SourceAccountID:      &defaultAccounts[0].ID,
			DestinationAccountID: &defaultAccounts[1].ID,
			Amount:               big.NewInt(150),
			Asset:                "USD/2",
			Metadata: map[string]string{
				"foo": "bar",
			},
		},
		{
			ID:                   piID3,
			ConnectorID:          defaultConnector.ID,
			Reference:            "test3",
			CreatedAt:            now.Add(-55 * time.Minute).UTC().Time,
			ScheduledAt:          now.Add(-40 * time.Minute).UTC().Time,
			Description:          "test3",
			Type:                 models.PAYMENT_INITIATION_TYPE_PAYOUT,
			DestinationAccountID: &defaultAccounts[1].ID,
			Amount:               big.NewInt(200),
			Asset:                "EUR/2",
			Metadata: map[string]string{
				"foo2": "bar2",
			},
		},
	}
}

func upsertPaymentInitiations(t *testing.T, ctx context.Context, storage Storage, paymentInitiations []models.PaymentInitiation) {
	for _, pi := range paymentInitiations {
		err := storage.PaymentInitiationsInsert(ctx, pi)
		require.NoError(t, err)
	}
}

func TestPaymentInitiationsInsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())

	t.Run("upsert with unknown connector", func(t *testing.T) {
		connector := models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}
		p := defaultPaymentInitiations()[0]
		p.ID.ConnectorID = connector
		p.ConnectorID = connector

		err := store.PaymentInitiationsInsert(ctx, p)
		require.Error(t, err)
	})

	t.Run("attempt insert with same id", func(t *testing.T) {
		defaultAccounts := defaultAccounts()
		pi := models.PaymentInitiation{
			ID:                   piID1,
			ConnectorID:          defaultConnector.ID,
			Reference:            "test_changed",
			CreatedAt:            now.Add(-30 * time.Minute).UTC().Time,
			ScheduledAt:          now.Add(-20 * time.Minute).UTC().Time,
			Description:          "test_changed",
			Type:                 models.PAYMENT_INITIATION_TYPE_PAYOUT,
			DestinationAccountID: &defaultAccounts[0].ID,
			Amount:               big.NewInt(100),
			Asset:                "DKK/2",
		}

		err := store.PaymentInitiationsInsert(ctx, pi)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrDuplicateKeyValue)

		actual, err := store.PaymentInitiationsGet(ctx, piID1)
		require.NoError(t, err)
		comparePaymentInitiations(t, defaultPaymentInitiations()[0], *actual)
	})

	t.Run("outbox event created for payment initiation", func(t *testing.T) {
		// Create a new payment initiation for this test
		defaultAccounts := defaultAccounts()
		newPI := models.PaymentInitiation{
			ID: models.PaymentInitiationID{
				Reference:   "outbox-test-pi",
				ConnectorID: defaultConnector.ID,
			},
			ConnectorID:          defaultConnector.ID,
			Reference:            "outbox-test-pi",
			CreatedAt:            now.Add(-10 * time.Minute).UTC().Time,
			ScheduledAt:          now.Add(-5 * time.Minute).UTC().Time,
			Description:          "Test Payment Initiation",
			Type:                 models.PAYMENT_INITIATION_TYPE_PAYOUT,
			DestinationAccountID: &defaultAccounts[0].ID,
			Amount:               big.NewInt(1000),
			Asset:                "USD/2",
			Metadata: map[string]string{
				"test": "outbox",
			},
		}

		expectedKey := newPI.IdempotencyKey()

		require.NoError(t, store.PaymentInitiationsInsert(ctx, newPI))

		// Verify outbox event was created
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		// Find our event
		var ourEvent *models.OutboxEvent
		for i := range pendingEvents {
			if pendingEvents[i].EventType == events.EventTypeSavedPaymentInitiation &&
				pendingEvents[i].EntityID == newPI.ID.String() &&
				pendingEvents[i].IdempotencyKey == expectedKey {
				ourEvent = &pendingEvents[i]
				break
			}
		}
		require.NotNil(t, ourEvent, "expected outbox event for payment initiation saved")

		// Verify event details
		assert.Equal(t, events.EventTypeSavedPaymentInitiation, ourEvent.EventType)
		assert.Equal(t, models.OUTBOX_STATUS_PENDING, ourEvent.Status)
		assert.Equal(t, newPI.ID.String(), ourEvent.EntityID)
		assert.Equal(t, newPI.ConnectorID, *ourEvent.ConnectorID)
		assert.Equal(t, 0, ourEvent.RetryCount)
		assert.Equal(t, expectedKey, ourEvent.IdempotencyKey)

		// Verify payload
		var payload map[string]interface{}
		err = json.Unmarshal(ourEvent.Payload, &payload)
		require.NoError(t, err)
		assert.Equal(t, newPI.ID.String(), payload["id"])
		assert.Equal(t, newPI.ConnectorID.String(), payload["connectorID"])
		assert.Equal(t, newPI.Reference, payload["reference"])
		assert.Equal(t, newPI.Amount.String(), payload["amount"])
		assert.Equal(t, newPI.Asset, payload["asset"])
	})
}

func TestPaymentInitiationsUpdateMetadata(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())

	t.Run("update metadata of unknown payment initiation", func(t *testing.T) {
		require.Error(t, store.PaymentInitiationsUpdateMetadata(ctx, models.PaymentInitiationID{
			Reference:   "unknown",
			ConnectorID: defaultConnector.ID,
		}, map[string]string{}))
	})

	t.Run("update existing metadata", func(t *testing.T) {
		metadata := map[string]string{
			"foo": "changed",
		}

		require.NoError(t, store.PaymentInitiationsUpdateMetadata(ctx, piID2, metadata))

		actual, err := store.PaymentInitiationsGet(ctx, piID2)
		require.NoError(t, err)
		require.Equal(t, len(metadata), len(actual.Metadata))
		for k, v := range metadata {
			_, ok := actual.Metadata[k]
			require.True(t, ok)
			require.Equal(t, v, actual.Metadata[k])
		}
	})

	t.Run("add new metadata", func(t *testing.T) {
		metadata := map[string]string{
			"key2": "value2",
			"key3": "value3",
		}

		require.NoError(t, store.PaymentInitiationsUpdateMetadata(ctx, piID1, metadata))

		actual, err := store.PaymentInitiationsGet(ctx, piID1)
		require.NoError(t, err)
		require.Equal(t, len(metadata), len(actual.Metadata))
		for k, v := range metadata {
			_, ok := actual.Metadata[k]
			require.True(t, ok)
			require.Equal(t, v, actual.Metadata[k])
		}
	})
}

func TestPaymentInitiationsGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())

	t.Run("get unknown payment initiation", func(t *testing.T) {
		_, err := store.PaymentInitiationsGet(ctx, models.PaymentInitiationID{
			Reference:   "unknown",
			ConnectorID: defaultConnector.ID,
		})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("get existing payment initiation", func(t *testing.T) {
		for _, pi := range defaultPaymentInitiations() {
			actual, err := store.PaymentInitiationsGet(ctx, pi.ID)
			require.NoError(t, err)
			comparePaymentInitiations(t, pi, *actual)
		}
	})
}

func TestPaymentInitiationsDelete(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())

	t.Run("delete unknown payment initiation", func(t *testing.T) {
		require.NoError(t, store.PaymentInitiationsDelete(ctx, models.PaymentInitiationID{
			Reference:   "unknown",
			ConnectorID: defaultConnector.ID,
		}))
	})

	t.Run("delete existing payment initiation", func(t *testing.T) {
		for _, pi := range defaultPaymentInitiations() {
			require.NoError(t, store.PaymentInitiationsDelete(ctx, pi.ID))

			_, err := store.PaymentInitiationsGet(ctx, pi.ID)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrNotFound)
		}
	})
}

func TestPaymentInitiationsDeleteFromConnectorID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())

	t.Run("delete from unknown connector", func(t *testing.T) {
		require.NoError(t, store.PaymentInitiationsDeleteFromConnectorID(ctx, models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}))

		for _, pi := range defaultPaymentInitiations() {
			actual, err := store.PaymentInitiationsGet(ctx, pi.ID)
			require.NoError(t, err)
			comparePaymentInitiations(t, pi, *actual)
		}
	})

	t.Run("delete from existing connector", func(t *testing.T) {
		require.NoError(t, store.PaymentInitiationsDeleteFromConnectorID(ctx, defaultConnector.ID))

		for _, pi := range defaultPaymentInitiations() {
			_, err := store.PaymentInitiationsGet(ctx, pi.ID)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrNotFound)
		}
	})
}

func TestPaymentInitiationsList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())
	upsertPaymentInitiationAdjustments(t, ctx, store, defaultPaymentInitiationAdjustments())

	t.Run("wrong query builder operator when listing by reference", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("reference", "test1")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
		assert.True(t, errors.Is(err, ErrValidation))
		assert.Regexp(t, "reference", err.Error())
	})

	t.Run("list payment intitiations by reference", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("reference", "test1")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePaymentInitiations(t, defaultPaymentInitiations()[0], cursor.Data[0])
	})

	t.Run("list payment initiations by unknown reference", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("reference", "unknown")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payment intitiations by id", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", defaultPaymentInitiations()[0].ID.String())),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePaymentInitiations(t, defaultPaymentInitiations()[0], cursor.Data[0])
	})

	t.Run("list payment initiations by unknown id", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", "unknown")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payment initiations by connector_id", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connector_id", defaultConnector.ID.String())),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 3)
		require.False(t, cursor.HasMore)
		comparePaymentInitiations(t, defaultPaymentInitiations()[1], cursor.Data[0])
		comparePaymentInitiations(t, defaultPaymentInitiations()[2], cursor.Data[1])
		comparePaymentInitiations(t, defaultPaymentInitiations()[0], cursor.Data[2])
	})

	t.Run("list payment initiations by unknown connector_id", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connector_id", "unknown")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payment initiations by type", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("type", "PAYOUT")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		comparePaymentInitiations(t, defaultPaymentInitiations()[2], cursor.Data[0])
		comparePaymentInitiations(t, defaultPaymentInitiations()[0], cursor.Data[1])
	})

	t.Run("list payment initiations by type 2", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("type", "TRANSFER")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePaymentInitiations(t, defaultPaymentInitiations()[1], cursor.Data[0])
	})

	t.Run("list payment initiations by unknown type", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("type", "UNKNOWN")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payment initiations by status multiple adjustments", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("status", models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED.String())),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePaymentInitiations(t, defaultPaymentInitiations()[0], cursor.Data[0])
	})

	t.Run("list payment initiations by status single adjustment", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("status", models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING.String())),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePaymentInitiations(t, defaultPaymentInitiations()[1], cursor.Data[0])
	})

	t.Run("list payment initiations by unknown status", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("status", "UNKNOWN")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payment initiations by asset", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("asset", "EUR/2")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		comparePaymentInitiations(t, defaultPaymentInitiations()[2], cursor.Data[0])
		comparePaymentInitiations(t, defaultPaymentInitiations()[0], cursor.Data[1])
	})

	t.Run("list payment initiations by asset 2", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("asset", "USD/2")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePaymentInitiations(t, defaultPaymentInitiations()[1], cursor.Data[0])
	})

	t.Run("list payment initiations by unknown asset", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("asset", "unknown")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payment initiations by source account id", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("source_account_id", defaultAccounts()[0].ID.String())),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePaymentInitiations(t, defaultPaymentInitiations()[1], cursor.Data[0])
	})

	t.Run("list payment initiations by unknown source account id", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("source_account_id", "unknown")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payment initiations by destination account id", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("destination_account_id", defaultAccounts()[1].ID.String())),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		comparePaymentInitiations(t, defaultPaymentInitiations()[1], cursor.Data[0])
		comparePaymentInitiations(t, defaultPaymentInitiations()[2], cursor.Data[1])
	})

	t.Run("list payment initiations by unknown destination account id", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("destination_account_id", "unknown")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payment initiations by amount", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("amount", 200)),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePaymentInitiations(t, defaultPaymentInitiations()[2], cursor.Data[0])
	})

	t.Run("list payment initiations by unknown amount", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("amount", 0)),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("wrong query builder operator when listing by metadata", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("metadata[foo]", "bar")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list payment initiations by metadata", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[foo]", "bar")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePaymentInitiations(t, defaultPaymentInitiations()[1], cursor.Data[0])
	})

	t.Run("list payment initiations by unknown metadata", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[foo]", "unknown")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payment initiations by unknown metadata 2", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[unknown]", "bar")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("unknown query builder key when listing", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("unknown", "test1")),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list payment initiations test cursor", func(t *testing.T) {
		q := NewListPaymentInitiationsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationQuery{}).
				WithPageSize(1),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePaymentInitiations(t, defaultPaymentInitiations()[1], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePaymentInitiations(t, defaultPaymentInitiations()[2], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		comparePaymentInitiations(t, defaultPaymentInitiations()[0], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePaymentInitiations(t, defaultPaymentInitiations()[2], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePaymentInitiations(t, defaultPaymentInitiations()[1], cursor.Data[0])
	})
}

func upsertPaymentInitiationRelatedPayments(t *testing.T, ctx context.Context, storage Storage) {
	payments := defaultPayments()
	require.NoError(t, storage.PaymentInitiationRelatedPaymentsUpsert(ctx, piID1, payments[0].ID, now.Add(-10*time.Minute).UTC().Time))
	require.NoError(t, storage.PaymentInitiationRelatedPaymentsUpsert(ctx, piID1, payments[1].ID, now.Add(-5*time.Minute).UTC().Time))
	require.NoError(t, storage.PaymentInitiationRelatedPaymentsUpsert(ctx, piID1, payments[2].ID, now.Add(-7*time.Minute).UTC().Time))
	require.NoError(t, storage.PaymentInitiationRelatedPaymentsUpsert(ctx, piID2, payments[0].ID, now.Add(-7*time.Minute).UTC().Time))
}

func TestPaymentInitiationsRelatedPaymentUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())
	upsertPaymentInitiationRelatedPayments(t, ctx, store)

	t.Run("same id insert", func(t *testing.T) {
		payments := defaultPayments()
		require.NoError(t, store.PaymentInitiationRelatedPaymentsUpsert(ctx, piID1, payments[0].ID, now.Add(-10*time.Minute).UTC().Time))

		cursor, err := store.PaymentInitiationRelatedPaymentsList(
			ctx,
			piID1,
			NewListPaymentInitiationRelatedPaymentsQuery(
				bunpaginate.NewPaginatedQueryOptions(PaymentInitiationRelatedPaymentsQuery{}).
					WithPageSize(15),
			),
		)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 3)
		require.False(t, cursor.HasMore)
		comparePayments(t, payments[1], cursor.Data[0])
		comparePayments(t, payments[2], cursor.Data[1])
		comparePayments(t, payments[0], cursor.Data[2])
	})

	t.Run("outbox event created for new related payment", func(t *testing.T) {
		// Clean up outbox events before test
		defer cleanupOutboxHelper(ctx, store)()

		// Create a new related payment for this test
		payments := defaultPayments()
		newPI := models.PaymentInitiation{
			ID: models.PaymentInitiationID{
				Reference:   "outbox-test-pi-related",
				ConnectorID: defaultConnector.ID,
			},
			ConnectorID: defaultConnector.ID,
			Reference:   "outbox-test-pi-related",
			CreatedAt:   now.Add(-10 * time.Minute).UTC().Time,
			ScheduledAt: now.Add(-5 * time.Minute).UTC().Time,
			Type:        models.PAYMENT_INITIATION_TYPE_PAYOUT,
			Amount:      big.NewInt(2000),
			Asset:       "EUR/2",
		}
		require.NoError(t, store.PaymentInitiationsInsert(ctx, newPI))

		relatedPayment := models.PaymentInitiationRelatedPayments{
			PaymentInitiationID: newPI.ID,
			PaymentID:           payments[0].ID,
		}
		expectedKey := relatedPayment.IdempotencyKey()

		require.NoError(t, store.PaymentInitiationRelatedPaymentsUpsert(ctx, newPI.ID, payments[0].ID, now.Add(-1*time.Minute).UTC().Time))

		// Verify outbox event was created
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		// Find our event
		var ourEvent *models.OutboxEvent
		for i := range pendingEvents {
			if pendingEvents[i].EventType == events.EventTypeSavedPaymentInitiationRelatedPayment &&
				pendingEvents[i].EntityID == fmt.Sprintf("%s:%s", newPI.ID.String(), payments[0].ID.String()) &&
				pendingEvents[i].IdempotencyKey == expectedKey {
				ourEvent = &pendingEvents[i]
				break
			}
		}
		require.NotNil(t, ourEvent, "expected outbox event for payment initiation related payment saved")

		// Verify event details
		assert.Equal(t, events.EventTypeSavedPaymentInitiationRelatedPayment, ourEvent.EventType)
		assert.Equal(t, models.OUTBOX_STATUS_PENDING, ourEvent.Status)
		assert.Equal(t, fmt.Sprintf("%s:%s", newPI.ID.String(), payments[0].ID.String()), ourEvent.EntityID)
		assert.Equal(t, newPI.ConnectorID, *ourEvent.ConnectorID)
		assert.Equal(t, 0, ourEvent.RetryCount)
		assert.Equal(t, expectedKey, ourEvent.IdempotencyKey)

		// Verify payload
		var payload map[string]interface{}
		err = json.Unmarshal(ourEvent.Payload, &payload)
		require.NoError(t, err)
		assert.Equal(t, newPI.ID.String(), payload["paymentInitiationID"])
		assert.Equal(t, payments[0].ID.String(), payload["paymentID"])
	})

	t.Run("no outbox event for existing related payment", func(t *testing.T) {
		// Clean up outbox events before test
		defer cleanupOutboxHelper(ctx, store)()

		// Count events before
		eventsBefore, err := store.OutboxEventsPollPending(ctx, 1000)
		require.NoError(t, err)
		countBefore := len(eventsBefore)

		// Insert existing related payment (should not create event)
		payments := defaultPayments()
		require.NoError(t, store.PaymentInitiationRelatedPaymentsUpsert(ctx, piID1, payments[0].ID, now.Add(-10*time.Minute).UTC().Time))

		// Verify no new outbox event was created
		eventsAfter, err := store.OutboxEventsPollPending(ctx, 1000)
		require.NoError(t, err)
		countAfter := len(eventsAfter)

		// Should have same number of events (no new related payment saved event)
		assert.Equal(t, countBefore, countAfter, "updating existing related payment should not create saved event")
	})

	t.Run("unknown payment initiation", func(t *testing.T) {
		payments := defaultPayments()
		require.Error(t, store.PaymentInitiationRelatedPaymentsUpsert(
			ctx,
			models.PaymentInitiationID{},
			payments[0].ID, now.Add(-10*time.Minute).UTC().Time),
		)
	})
}

func TestPaymentInitiationIDsFromPaymentID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())
	upsertPaymentInitiationRelatedPayments(t, ctx, store)

	t.Run("unknown payment id", func(t *testing.T) {
		ids, err := store.PaymentInitiationIDsListFromPaymentID(ctx, models.PaymentID{})
		require.NoError(t, err)
		require.Len(t, ids, 0)
	})

	t.Run("known payment id", func(t *testing.T) {
		ids, err := store.PaymentInitiationIDsListFromPaymentID(ctx, defaultPayments()[0].ID)
		require.NoError(t, err)
		require.Len(t, ids, 2)
		require.Contains(t, ids, piID1)
		require.Contains(t, ids, piID2)
	})
}

func TestPaymentInitiationRelatedPaymentsList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())
	upsertPaymentInitiationRelatedPayments(t, ctx, store)

	t.Run("list related payments by unknown payment initiation", func(t *testing.T) {
		cursor, err := store.PaymentInitiationRelatedPaymentsList(
			ctx,
			models.PaymentInitiationID{},
			NewListPaymentInitiationRelatedPaymentsQuery(
				bunpaginate.NewPaginatedQueryOptions(PaymentInitiationRelatedPaymentsQuery{}).
					WithPageSize(15),
			),
		)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list related payments by payment initiation", func(t *testing.T) {
		q := NewListPaymentInitiationRelatedPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationRelatedPaymentsQuery{}).
				WithPageSize(1),
		)
		payments := defaultPayments()

		cursor, err := store.PaymentInitiationRelatedPaymentsList(ctx, piID1, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePayments(t, payments[1], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationRelatedPaymentsList(ctx, piID1, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePayments(t, payments[2], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationRelatedPaymentsList(ctx, piID1, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		comparePayments(t, payments[0], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationRelatedPaymentsList(ctx, piID1, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePayments(t, payments[2], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationRelatedPaymentsList(ctx, piID1, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePayments(t, payments[1], cursor.Data[0])
	})
}

var (
	piAdjID1 = models.PaymentInitiationAdjustmentID{
		PaymentInitiationID: defaultPaymentInitiations()[0].ID,
		CreatedAt:           now.Add(-10 * time.Minute).UTC().Time,
		Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION,
	}
	piAdjID2 = models.PaymentInitiationAdjustmentID{
		PaymentInitiationID: defaultPaymentInitiations()[0].ID,
		CreatedAt:           now.Add(-5 * time.Minute).UTC().Time,
		Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
	}
	piAdjID3 = models.PaymentInitiationAdjustmentID{
		PaymentInitiationID: defaultPaymentInitiations()[1].ID,
		CreatedAt:           now.Add(-7 * time.Minute).UTC().Time,
		Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING,
	}
)

func defaultPaymentInitiationAdjustments() []models.PaymentInitiationAdjustment {
	return []models.PaymentInitiationAdjustment{
		{
			ID:        piAdjID1,
			CreatedAt: now.Add(-10 * time.Minute).UTC().Time,
			Status:    models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION,
			Amount:    big.NewInt(100),
			Asset:     pointer.For("EUR/2"),
			Metadata: map[string]string{
				"foo": "bar",
			},
		},
		{
			ID:        piAdjID2,
			CreatedAt: now.Add(-5 * time.Minute).UTC().Time,
			Status:    models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
			Error:     errors.New("test"),
			Amount:    big.NewInt(200),
			Asset:     pointer.For("USD/2"),
			Metadata: map[string]string{
				"foo2": "bar2",
			},
		},
		{
			ID:        piAdjID3,
			CreatedAt: now.Add(-7 * time.Minute).UTC().Time,
			Status:    models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING,
			Amount:    big.NewInt(300),
			Asset:     pointer.For("DKK/2"),
			Metadata: map[string]string{
				"foo3": "bar3",
			},
		},
	}
}

func upsertPaymentInitiationAdjustments(t *testing.T, ctx context.Context, storage Storage, adjustments []models.PaymentInitiationAdjustment) {
	for _, adj := range adjustments {
		require.NoError(t, storage.PaymentInitiationAdjustmentsUpsert(ctx, adj))
	}
}

func TestPaymentInitiationAdjustmentsUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())
	upsertPaymentInitiationAdjustments(t, ctx, store, defaultPaymentInitiationAdjustments())

	t.Run("upsert with unknown payment initiation", func(t *testing.T) {
		p := models.PaymentInitiationAdjustment{
			ID:        models.PaymentInitiationAdjustmentID{},
			CreatedAt: now.Add(-10 * time.Minute).UTC().Time,
			Status:    models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION,
			Metadata: map[string]string{
				"foo": "bar",
			},
		}

		require.Error(t, store.PaymentInitiationAdjustmentsUpsert(ctx, p))
	})

	t.Run("upsert with same id", func(t *testing.T) {
		p := models.PaymentInitiationAdjustment{
			ID:        piAdjID1,
			CreatedAt: now.Add(-30 * time.Minute).UTC().Time,
			Status:    models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
			Metadata: map[string]string{
				"foo": "changed",
			},
		}

		require.NoError(t, store.PaymentInitiationAdjustmentsUpsert(ctx, p))

		for _, pa := range defaultPaymentInitiationAdjustments() {
			actual, err := store.PaymentInitiationAdjustmentsGet(ctx, pa.ID)
			require.NoError(t, err)
			comparePaymentInitiationAdjustments(t, pa, *actual)
		}
	})

	t.Run("outbox event created for new adjustment", func(t *testing.T) {
		// Clean up outbox events before test
		defer cleanupOutboxHelper(ctx, store)()

		// Create a new adjustment for this test
		newAdj := models.PaymentInitiationAdjustment{
			ID: models.PaymentInitiationAdjustmentID{
				PaymentInitiationID: defaultPaymentInitiations()[0].ID,
				CreatedAt:           now.Add(-2 * time.Minute).UTC().Time,
				Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING,
			},
			CreatedAt: now.Add(-2 * time.Minute).UTC().Time,
			Status:    models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING,
			Amount:    big.NewInt(500),
			Asset:     pointer.For("GBP/2"),
			Metadata: map[string]string{
				"test": "outbox",
			},
		}

		expectedKey := newAdj.IdempotencyKey()

		require.NoError(t, store.PaymentInitiationAdjustmentsUpsert(ctx, newAdj))

		// Verify outbox event was created
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		// Find our event
		var ourEvent *models.OutboxEvent
		for i := range pendingEvents {
			if pendingEvents[i].EventType == events.EventTypeSavedPaymentInitiationAdjustment &&
				pendingEvents[i].EntityID == newAdj.ID.String() &&
				pendingEvents[i].IdempotencyKey == expectedKey {
				ourEvent = &pendingEvents[i]
				break
			}
		}
		require.NotNil(t, ourEvent, "expected outbox event for payment initiation adjustment saved")

		// Verify event details
		assert.Equal(t, events.EventTypeSavedPaymentInitiationAdjustment, ourEvent.EventType)
		assert.Equal(t, models.OUTBOX_STATUS_PENDING, ourEvent.Status)
		assert.Equal(t, newAdj.ID.String(), ourEvent.EntityID)
		assert.Equal(t, newAdj.ID.PaymentInitiationID.ConnectorID, *ourEvent.ConnectorID)
		assert.Equal(t, 0, ourEvent.RetryCount)
		assert.Equal(t, expectedKey, ourEvent.IdempotencyKey)

		// Verify payload
		var payload map[string]interface{}
		err = json.Unmarshal(ourEvent.Payload, &payload)
		require.NoError(t, err)
		assert.Equal(t, newAdj.ID.String(), payload["id"])
		assert.Equal(t, newAdj.ID.PaymentInitiationID.String(), payload["paymentInitiationID"])
		assert.Equal(t, newAdj.Status.String(), payload["status"])
		assert.Equal(t, newAdj.Amount.String(), payload["amount"])
		assert.Equal(t, *newAdj.Asset, payload["asset"])
		// Metadata is unmarshaled as map[string]interface{}, so we need to compare values
		payloadMetadata, ok := payload["metadata"].(map[string]interface{})
		require.True(t, ok, "metadata should be a map")
		assert.Equal(t, newAdj.Metadata["test"], payloadMetadata["test"])
	})

	t.Run("no outbox event for existing adjustment update", func(t *testing.T) {
		// Clean up outbox events before test
		defer cleanupOutboxHelper(ctx, store)()

		// Count events before
		eventsBefore, err := store.OutboxEventsPollPending(ctx, 1000)
		require.NoError(t, err)
		countBefore := len(eventsBefore)

		// Update existing adjustment (should not create event)
		existingAdj := defaultPaymentInitiationAdjustments()[0]
		existingAdj.Metadata = map[string]string{"updated": "true"}
		require.NoError(t, store.PaymentInitiationAdjustmentsUpsert(ctx, existingAdj))

		// Verify no new outbox event was created
		eventsAfter, err := store.OutboxEventsPollPending(ctx, 1000)
		require.NoError(t, err)
		countAfter := len(eventsAfter)

		// Should have same number of events (no new adjustment saved event)
		assert.Equal(t, countBefore, countAfter, "updating existing adjustment should not create saved event")
	})
}

func TestPaymentInitiationAdjustmentsUpsertIfStatusEqual(t *testing.T) {
	t.Parallel()
	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())
	upsertPaymentInitiationAdjustments(t, ctx, store, defaultPaymentInitiationAdjustments())
	t.Run("upsert with status not equal", func(t *testing.T) {
		p := models.PaymentInitiationAdjustment{
			ID: models.PaymentInitiationAdjustmentID{
				PaymentInitiationID: defaultPaymentInitiations()[1].ID,
				CreatedAt:           now.UTC().Time,
				Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING,
			},
			CreatedAt: now.UTC().Time,
			Status:    models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING,
			Metadata: map[string]string{
				"foo": "bar",
			},
		}
		inserted, err := store.PaymentInitiationAdjustmentsUpsertIfPredicate(
			ctx,
			p,
			func(previous models.PaymentInitiationAdjustment) bool {
				return previous.Status == models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION
			},
		)
		require.NoError(t, err)
		require.False(t, inserted)
	})
	t.Run("upsert with status equal", func(t *testing.T) {
		// Clean up outbox events before test
		defer cleanupOutboxHelper(ctx, store)()

		p := models.PaymentInitiationAdjustment{
			ID: models.PaymentInitiationAdjustmentID{
				PaymentInitiationID: defaultPaymentInitiations()[0].ID,
				CreatedAt:           now.UTC().Time,
				Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING,
			},
			CreatedAt: now.UTC().Time,
			Status:    models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING,
			Amount:    big.NewInt(1000),
			Asset:     pointer.For("USD/2"),
			Metadata: map[string]string{
				"foo": "bar",
			},
		}
		expectedKey := p.IdempotencyKey()

		inserted, err := store.PaymentInitiationAdjustmentsUpsertIfPredicate(
			ctx,
			p,
			func(previous models.PaymentInitiationAdjustment) bool {
				return previous.Status == models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED
			},
		)
		require.NoError(t, err)
		require.True(t, inserted)

		// Verify outbox event was created
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		// Find our event
		var ourEvent *models.OutboxEvent
		for i := range pendingEvents {
			if pendingEvents[i].EventType == events.EventTypeSavedPaymentInitiationAdjustment &&
				pendingEvents[i].EntityID == p.ID.String() &&
				pendingEvents[i].IdempotencyKey == expectedKey {
				ourEvent = &pendingEvents[i]
				break
			}
		}
		require.NotNil(t, ourEvent, "expected outbox event for payment initiation adjustment saved")

		// Verify event details
		assert.Equal(t, events.EventTypeSavedPaymentInitiationAdjustment, ourEvent.EventType)
		assert.Equal(t, models.OUTBOX_STATUS_PENDING, ourEvent.Status)
		assert.Equal(t, p.ID.String(), ourEvent.EntityID)
		assert.Equal(t, p.ID.PaymentInitiationID.ConnectorID, *ourEvent.ConnectorID)
		assert.Equal(t, 0, ourEvent.RetryCount)
		assert.Equal(t, expectedKey, ourEvent.IdempotencyKey)

		// Verify payload
		var payload map[string]interface{}
		err = json.Unmarshal(ourEvent.Payload, &payload)
		require.NoError(t, err)
		assert.Equal(t, p.ID.String(), payload["id"])
		assert.Equal(t, p.ID.PaymentInitiationID.String(), payload["paymentInitiationID"])
		assert.Equal(t, p.Status.String(), payload["status"])
		assert.Equal(t, p.Amount.String(), payload["amount"])
		assert.Equal(t, *p.Asset, payload["asset"])
	})

	t.Run("no outbox event when predicate fails", func(t *testing.T) {
		// Clean up outbox events before test
		defer cleanupOutboxHelper(ctx, store)()

		// Count events before
		eventsBefore, err := store.OutboxEventsPollPending(ctx, 1000)
		require.NoError(t, err)
		countBefore := len(eventsBefore)

		p := models.PaymentInitiationAdjustment{
			ID: models.PaymentInitiationAdjustmentID{
				PaymentInitiationID: defaultPaymentInitiations()[1].ID,
				CreatedAt:           now.UTC().Time,
				Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING,
			},
			CreatedAt: now.UTC().Time,
			Status:    models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING,
			Metadata: map[string]string{
				"foo": "bar",
			},
		}
		inserted, err := store.PaymentInitiationAdjustmentsUpsertIfPredicate(
			ctx,
			p,
			func(previous models.PaymentInitiationAdjustment) bool {
				return previous.Status == models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION
			},
		)
		require.NoError(t, err)
		require.False(t, inserted)

		// Verify no new outbox event was created
		eventsAfter, err := store.OutboxEventsPollPending(ctx, 1000)
		require.NoError(t, err)
		countAfter := len(eventsAfter)

		// Should have same number of events (no new adjustment saved event)
		assert.Equal(t, countBefore, countAfter, "predicate failure should not create saved event")
	})
}

func TestPaymentInitiationAdjustmentsGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())
	upsertPaymentInitiationAdjustments(t, ctx, store, defaultPaymentInitiationAdjustments())

	t.Run("get unknown payment initiation adjustment", func(t *testing.T) {
		_, err := store.PaymentInitiationAdjustmentsGet(ctx, models.PaymentInitiationAdjustmentID{})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("get existing payment initiation adjustment", func(t *testing.T) {
		for _, pa := range defaultPaymentInitiationAdjustments() {
			actual, err := store.PaymentInitiationAdjustmentsGet(ctx, pa.ID)
			require.NoError(t, err)
			comparePaymentInitiationAdjustments(t, pa, *actual)
		}
	})
}

func TestPaymentInitiationAdjustmentsList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())
	upsertPaymentInitiationAdjustments(t, ctx, store, defaultPaymentInitiationAdjustments())

	t.Run("list payment initiation adjustments by unknown payment initiation", func(t *testing.T) {
		cursor, err := store.PaymentInitiationAdjustmentsList(
			ctx,
			models.PaymentInitiationID{},
			NewListPaymentInitiationAdjustmentsQuery(
				bunpaginate.NewPaginatedQueryOptions(PaymentInitiationAdjustmentsQuery{}).
					WithPageSize(15),
			),
		)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("wrong query builder operator when listing by status", func(t *testing.T) {
		q := NewListPaymentInitiationAdjustmentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationAdjustmentsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("status", models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED)),
		)

		cursor, err := store.PaymentInitiationAdjustmentsList(ctx, defaultPaymentInitiationAdjustments()[0].ID.PaymentInitiationID, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list payment initiation adjustments by status", func(t *testing.T) {
		q := NewListPaymentInitiationAdjustmentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationAdjustmentsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("status", models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED)),
		)

		cursor, err := store.PaymentInitiationAdjustmentsList(ctx, defaultPaymentInitiationAdjustments()[0].ID.PaymentInitiationID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePaymentInitiationAdjustments(t, defaultPaymentInitiationAdjustments()[1], cursor.Data[0])
	})

	t.Run("wrong query builder operator when listing by metadata", func(t *testing.T) {
		q := NewListPaymentInitiationAdjustmentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationAdjustmentsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("metadata[foo]", "bar")),
		)

		cursor, err := store.PaymentInitiationAdjustmentsList(ctx, defaultPaymentInitiationAdjustments()[0].ID.PaymentInitiationID, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list payment initiation adjustments by metadata", func(t *testing.T) {
		q := NewListPaymentInitiationAdjustmentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationAdjustmentsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[foo]", "bar")),
		)

		cursor, err := store.PaymentInitiationAdjustmentsList(ctx, defaultPaymentInitiationAdjustments()[0].ID.PaymentInitiationID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePaymentInitiationAdjustments(t, defaultPaymentInitiationAdjustments()[0], cursor.Data[0])
	})

	t.Run("unknown query builder key when listing", func(t *testing.T) {
		q := NewListPaymentInitiationAdjustmentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationAdjustmentsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("unknown", "test1")),
		)

		cursor, err := store.PaymentInitiationAdjustmentsList(ctx, defaultPaymentInitiationAdjustments()[0].ID.PaymentInitiationID, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list payment initiation adjustments by payment initiation", func(t *testing.T) {
		q := NewListPaymentInitiationAdjustmentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationAdjustmentsQuery{}).
				WithPageSize(1),
		)

		cursor, err := store.PaymentInitiationAdjustmentsList(ctx, defaultPaymentInitiationAdjustments()[0].ID.PaymentInitiationID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePaymentInitiationAdjustments(t, defaultPaymentInitiationAdjustments()[1], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationAdjustmentsList(ctx, defaultPaymentInitiationAdjustments()[0].ID.PaymentInitiationID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		comparePaymentInitiationAdjustments(t, defaultPaymentInitiationAdjustments()[0], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationAdjustmentsList(ctx, defaultPaymentInitiationAdjustments()[0].ID.PaymentInitiationID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePaymentInitiationAdjustments(t, defaultPaymentInitiationAdjustments()[1], cursor.Data[0])
	})
}

func comparePaymentInitiations(t *testing.T, expected, actual models.PaymentInitiation) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.ConnectorID, actual.ConnectorID)
	require.Equal(t, expected.Reference, actual.Reference)
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
	require.Equal(t, expected.ScheduledAt, actual.ScheduledAt)
	require.Equal(t, expected.Description, actual.Description)
	require.Equal(t, expected.Type, actual.Type)

	switch {
	case expected.SourceAccountID != nil && actual.SourceAccountID != nil:
		require.Equal(t, *expected.SourceAccountID, *actual.SourceAccountID)
	case expected.SourceAccountID == nil && actual.SourceAccountID == nil:
	default:
		t.Fatalf("expected.SourceAccountID != actual.SourceAccountID")
	}

	require.Equal(t, expected.DestinationAccountID, actual.DestinationAccountID)
	require.Equal(t, expected.Amount, actual.Amount)
	require.Equal(t, expected.Asset, actual.Asset)

	require.Equal(t, len(expected.Metadata), len(actual.Metadata))
	for k, v := range expected.Metadata {
		_, ok := actual.Metadata[k]
		require.True(t, ok)
		require.Equal(t, v, actual.Metadata[k])
	}
}

func comparePaymentInitiationAdjustments(t *testing.T, expected, actual models.PaymentInitiationAdjustment) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
	require.Equal(t, expected.Status, actual.Status)

	switch {
	case expected.Error != nil && actual.Error != nil:
		require.Equal(t, expected.Error.Error(), actual.Error.Error())
	case expected.Error == nil && actual.Error == nil:
	default:
		t.Fatalf("expected.Error != actual.Error")
	}

	require.Equal(t, len(expected.Metadata), len(actual.Metadata))
	for k, v := range expected.Metadata {
		_, ok := actual.Metadata[k]
		require.True(t, ok)
		require.Equal(t, v, actual.Metadata[k])
	}
}
