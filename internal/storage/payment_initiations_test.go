package storage

import (
	"context"
	"math/big"
	"testing"

	"github.com/formancehq/go-libs/bun/bunpaginate"
	"github.com/formancehq/go-libs/logging"
	"github.com/formancehq/go-libs/pointer"
	"github.com/formancehq/go-libs/query"
	"github.com/formancehq/go-libs/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
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

	defaultPaymentInitiations = []models.PaymentInitiation{
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
)

func upsertPaymentInitiations(t *testing.T, ctx context.Context, storage Storage, paymentInitiations []models.PaymentInitiation) {
	for _, pi := range paymentInitiations {
		err := storage.PaymentInitiationsUpsert(ctx, pi)
		require.NoError(t, err)
	}
}

func TestPaymentInitiationsUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts)
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations)

	t.Run("upsert with unknown connector", func(t *testing.T) {
		connector := models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}
		p := defaultPaymentInitiations[0]
		p.ID.ConnectorID = connector
		p.ConnectorID = connector

		err := store.PaymentInitiationsUpsert(ctx, p)
		require.Error(t, err)
	})

	t.Run("upsert with same id", func(t *testing.T) {
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

		upsertPaymentInitiations(t, ctx, store, []models.PaymentInitiation{pi})

		actual, err := store.PaymentInitiationsGet(ctx, piID1)
		require.NoError(t, err)
		comparePaymentInitiations(t, defaultPaymentInitiations[0], *actual)
	})
}

func TestPaymentInitiationsUpdateMetadata(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts)
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations)

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

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts)
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations)

	t.Run("get unknown payment initiation", func(t *testing.T) {
		_, err := store.PaymentInitiationsGet(ctx, models.PaymentInitiationID{
			Reference:   "unknown",
			ConnectorID: defaultConnector.ID,
		})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("get existing payment initiation", func(t *testing.T) {
		for _, pi := range defaultPaymentInitiations {
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

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts)
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations)

	t.Run("delete unknown payment initiation", func(t *testing.T) {
		require.NoError(t, store.PaymentInitiationsDelete(ctx, models.PaymentInitiationID{
			Reference:   "unknown",
			ConnectorID: defaultConnector.ID,
		}))
	})

	t.Run("delete existing payment initiation", func(t *testing.T) {
		for _, pi := range defaultPaymentInitiations {
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

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts)
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations)

	t.Run("delete from unknown connector", func(t *testing.T) {
		require.NoError(t, store.PaymentInitiationsDeleteFromConnectorID(ctx, models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}))

		for _, pi := range defaultPaymentInitiations {
			actual, err := store.PaymentInitiationsGet(ctx, pi.ID)
			require.NoError(t, err)
			comparePaymentInitiations(t, pi, *actual)
		}
	})

	t.Run("delete from existing connector", func(t *testing.T) {
		require.NoError(t, store.PaymentInitiationsDeleteFromConnectorID(ctx, defaultConnector.ID))

		for _, pi := range defaultPaymentInitiations {
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

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts)
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations)

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
		comparePaymentInitiations(t, defaultPaymentInitiations[0], cursor.Data[0])
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
		comparePaymentInitiations(t, defaultPaymentInitiations[1], cursor.Data[0])
		comparePaymentInitiations(t, defaultPaymentInitiations[2], cursor.Data[1])
		comparePaymentInitiations(t, defaultPaymentInitiations[0], cursor.Data[2])
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
		comparePaymentInitiations(t, defaultPaymentInitiations[2], cursor.Data[0])
		comparePaymentInitiations(t, defaultPaymentInitiations[0], cursor.Data[1])
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
		comparePaymentInitiations(t, defaultPaymentInitiations[1], cursor.Data[0])
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
		comparePaymentInitiations(t, defaultPaymentInitiations[2], cursor.Data[0])
		comparePaymentInitiations(t, defaultPaymentInitiations[0], cursor.Data[1])
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
		comparePaymentInitiations(t, defaultPaymentInitiations[1], cursor.Data[0])
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
				WithQueryBuilder(query.Match("source_account_id", defaultAccounts[0].ID.String())),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePaymentInitiations(t, defaultPaymentInitiations[1], cursor.Data[0])
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
				WithQueryBuilder(query.Match("destination_account_id", defaultAccounts[1].ID.String())),
		)

		cursor, err := store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		comparePaymentInitiations(t, defaultPaymentInitiations[1], cursor.Data[0])
		comparePaymentInitiations(t, defaultPaymentInitiations[2], cursor.Data[1])
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
		comparePaymentInitiations(t, defaultPaymentInitiations[2], cursor.Data[0])
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
		comparePaymentInitiations(t, defaultPaymentInitiations[1], cursor.Data[0])
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
		comparePaymentInitiations(t, defaultPaymentInitiations[1], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePaymentInitiations(t, defaultPaymentInitiations[2], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		comparePaymentInitiations(t, defaultPaymentInitiations[0], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePaymentInitiations(t, defaultPaymentInitiations[2], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePaymentInitiations(t, defaultPaymentInitiations[1], cursor.Data[0])
	})
}

func upsertPaymentInitiationRelatedPayments(t *testing.T, ctx context.Context, storage Storage) {
	require.NoError(t, storage.PaymentInitiationRelatedPaymentsUpsert(ctx, piID1, defaultPayments[0].ID, now.Add(-10*time.Minute).UTC().Time))
	require.NoError(t, storage.PaymentInitiationRelatedPaymentsUpsert(ctx, piID1, defaultPayments[1].ID, now.Add(-5*time.Minute).UTC().Time))
	require.NoError(t, storage.PaymentInitiationRelatedPaymentsUpsert(ctx, piID1, defaultPayments[2].ID, now.Add(-7*time.Minute).UTC().Time))
	require.NoError(t, storage.PaymentInitiationRelatedPaymentsUpsert(ctx, piID2, defaultPayments[0].ID, now.Add(-7*time.Minute).UTC().Time))
}

func TestPaymentInitiationsRelatedPaymentUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts)
	upsertPayments(t, ctx, store, defaultPayments)
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations)
	upsertPaymentInitiationRelatedPayments(t, ctx, store)

	t.Run("same id insert", func(t *testing.T) {
		require.NoError(t, store.PaymentInitiationRelatedPaymentsUpsert(ctx, piID1, defaultPayments[0].ID, now.Add(-10*time.Minute).UTC().Time))

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
		comparePayments(t, defaultPayments[1], cursor.Data[0])
		comparePayments(t, defaultPayments[2], cursor.Data[1])
		comparePayments(t, defaultPayments[0], cursor.Data[2])
	})

	t.Run("unknown payment initiation", func(t *testing.T) {
		require.Error(t, store.PaymentInitiationRelatedPaymentsUpsert(
			ctx,
			models.PaymentInitiationID{},
			defaultPayments[0].ID, now.Add(-10*time.Minute).UTC().Time),
		)
	})

	t.Run("unknown payment id", func(t *testing.T) {
		require.Error(t, store.PaymentInitiationRelatedPaymentsUpsert(
			ctx,
			piID1,
			models.PaymentID{},
			now.Add(-10*time.Minute).UTC().Time),
		)
	})
}

func TestPaymentInitiationRelatedPaymentsList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts)
	upsertPayments(t, ctx, store, defaultPayments)
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations)
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

		cursor, err := store.PaymentInitiationRelatedPaymentsList(ctx, piID1, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePayments(t, defaultPayments[1], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationRelatedPaymentsList(ctx, piID1, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePayments(t, defaultPayments[2], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationRelatedPaymentsList(ctx, piID1, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		comparePayments(t, defaultPayments[0], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationRelatedPaymentsList(ctx, piID1, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePayments(t, defaultPayments[2], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationRelatedPaymentsList(ctx, piID1, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePayments(t, defaultPayments[1], cursor.Data[0])
	})
}

var (
	piAdjID1 = models.PaymentInitiationAdjustmentID{
		PaymentInitiationID: defaultPaymentInitiations[0].ID,
		CreatedAt:           now.Add(-10 * time.Minute).UTC().Time,
		Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION,
	}
	piAdjID2 = models.PaymentInitiationAdjustmentID{
		PaymentInitiationID: defaultPaymentInitiations[0].ID,
		CreatedAt:           now.Add(-5 * time.Minute).UTC().Time,
		Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
	}
	piAdjID3 = models.PaymentInitiationAdjustmentID{
		PaymentInitiationID: defaultPaymentInitiations[1].ID,
		CreatedAt:           now.Add(-7 * time.Minute).UTC().Time,
		Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING,
	}

	defaultPaymentInitiationAdjustments = []models.PaymentInitiationAdjustment{
		{
			ID:                  piAdjID1,
			PaymentInitiationID: defaultPaymentInitiations[0].ID,
			CreatedAt:           now.Add(-10 * time.Minute).UTC().Time,
			Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION,
			Metadata: map[string]string{
				"foo": "bar",
			},
		},
		{
			ID:                  piAdjID2,
			PaymentInitiationID: defaultPaymentInitiations[0].ID,
			CreatedAt:           now.Add(-5 * time.Minute).UTC().Time,
			Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
			Error:               pointer.For("test"),
			Metadata: map[string]string{
				"foo2": "bar2",
			},
		},
		{
			ID:                  piAdjID3,
			PaymentInitiationID: defaultPaymentInitiations[1].ID,
			CreatedAt:           now.Add(-7 * time.Minute).UTC().Time,
			Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING,
			Metadata: map[string]string{
				"foo3": "bar3",
			},
		},
	}
)

func upsertPaymentInitiationAdjustments(t *testing.T, ctx context.Context, storage Storage, adjustments []models.PaymentInitiationAdjustment) {
	for _, adj := range adjustments {
		require.NoError(t, storage.PaymentInitiationAdjustmentsUpsert(ctx, adj))
	}
}

func TestPaymentInitiationAdjustmentsUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts)
	upsertPayments(t, ctx, store, defaultPayments)
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations)
	upsertPaymentInitiationAdjustments(t, ctx, store, defaultPaymentInitiationAdjustments)

	t.Run("upsert with unknown payment initiation", func(t *testing.T) {
		p := models.PaymentInitiationAdjustment{
			ID:                  models.PaymentInitiationAdjustmentID{},
			PaymentInitiationID: models.PaymentInitiationID{},
			CreatedAt:           now.Add(-10 * time.Minute).UTC().Time,
			Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION,
			Metadata: map[string]string{
				"foo": "bar",
			},
		}

		require.Error(t, store.PaymentInitiationAdjustmentsUpsert(ctx, p))
	})

	t.Run("upsert with same id", func(t *testing.T) {
		p := models.PaymentInitiationAdjustment{
			ID:                  piAdjID1,
			PaymentInitiationID: defaultPaymentInitiationAdjustments[0].PaymentInitiationID,
			CreatedAt:           now.Add(-30 * time.Minute).UTC().Time,
			Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
			Metadata: map[string]string{
				"foo": "changed",
			},
		}

		require.NoError(t, store.PaymentInitiationAdjustmentsUpsert(ctx, p))

		for _, pa := range defaultPaymentInitiationAdjustments {
			actual, err := store.PaymentInitiationAdjustmentsGet(ctx, pa.ID)
			require.NoError(t, err)
			comparePaymentInitiationAdjustments(t, pa, *actual)
		}
	})
}

func TestPaymentInitiationAdjustmentsGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts)
	upsertPayments(t, ctx, store, defaultPayments)
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations)
	upsertPaymentInitiationAdjustments(t, ctx, store, defaultPaymentInitiationAdjustments)

	t.Run("get unknown payment initiation adjustment", func(t *testing.T) {
		_, err := store.PaymentInitiationAdjustmentsGet(ctx, models.PaymentInitiationAdjustmentID{})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("get existing payment initiation adjustment", func(t *testing.T) {
		for _, pa := range defaultPaymentInitiationAdjustments {
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

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts)
	upsertPayments(t, ctx, store, defaultPayments)
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations)
	upsertPaymentInitiationAdjustments(t, ctx, store, defaultPaymentInitiationAdjustments)

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

	t.Run("list payment initiation adjustments by payment initiation", func(t *testing.T) {
		q := NewListPaymentInitiationAdjustmentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationAdjustmentsQuery{}).
				WithPageSize(1),
		)

		cursor, err := store.PaymentInitiationAdjustmentsList(ctx, defaultPaymentInitiationAdjustments[0].PaymentInitiationID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePaymentInitiationAdjustments(t, defaultPaymentInitiationAdjustments[1], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationAdjustmentsList(ctx, defaultPaymentInitiationAdjustments[0].PaymentInitiationID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		comparePaymentInitiationAdjustments(t, defaultPaymentInitiationAdjustments[0], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationAdjustmentsList(ctx, defaultPaymentInitiationAdjustments[0].PaymentInitiationID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePaymentInitiationAdjustments(t, defaultPaymentInitiationAdjustments[1], cursor.Data[0])
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
	require.Equal(t, expected.PaymentInitiationID, actual.PaymentInitiationID)
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
	require.Equal(t, expected.Status, actual.Status)
	require.Equal(t, expected.Error, actual.Error)

	require.Equal(t, len(expected.Metadata), len(actual.Metadata))
	for k, v := range expected.Metadata {
		_, ok := actual.Metadata[k]
		require.True(t, ok)
		require.Equal(t, v, actual.Metadata[k])
	}
}
