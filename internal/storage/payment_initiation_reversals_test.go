package storage

import (
	"context"
	"github.com/formancehq/go-libs/v3/platform/postgres"
	"math/big"
	"testing"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	pirID1 = models.PaymentInitiationReversalID{
		Reference:   "test1",
		ConnectorID: defaultConnector.ID,
	}

	pirID2 = models.PaymentInitiationReversalID{
		Reference:   "test2",
		ConnectorID: defaultConnector.ID,
	}

	pirID3 = models.PaymentInitiationReversalID{
		Reference:   "test3",
		ConnectorID: defaultConnector.ID,
	}
)

func defaultPaymentInitiationReversals() []models.PaymentInitiationReversal {
	return []models.PaymentInitiationReversal{
		{
			ID:                  pirID1,
			ConnectorID:         defaultConnector.ID,
			PaymentInitiationID: piID1,
			Reference:           "test1",
			CreatedAt:           now.Add(-60 * time.Minute).UTC().Time,
			Description:         "test1",
			Amount:              big.NewInt(100),
			Asset:               "EUR/2",
			Metadata:            map[string]string{},
		},
		{
			ID:                  pirID2,
			ConnectorID:         defaultConnector.ID,
			PaymentInitiationID: piID1,
			Reference:           "test2",
			CreatedAt:           now.Add(-30 * time.Minute).UTC().Time,
			Description:         "test2",
			Amount:              big.NewInt(150),
			Asset:               "USD/2",
			Metadata:            map[string]string{"foo": "bar"},
		},
		{
			ID:                  pirID3,
			ConnectorID:         defaultConnector.ID,
			PaymentInitiationID: piID2,
			Reference:           "test3",
			CreatedAt:           now.Add(-55 * time.Minute).UTC().Time,
			Description:         "test3",
			Amount:              big.NewInt(200),
			Asset:               "EUR/2",
			Metadata:            map[string]string{"foo2": "bar2"},
		},
	}
}

func upsertPaymentInitiationReversals(t *testing.T, ctx context.Context, storage Storage, paymentInitiationReversals []models.PaymentInitiationReversal) {
	for _, pi := range paymentInitiationReversals {
		err := storage.PaymentInitiationReversalsUpsert(ctx, pi, nil)
		require.NoError(t, err)
	}
}

func TestPaymentInitiationReversalsUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())
	upsertPaymentInitiationReversals(t, ctx, store, defaultPaymentInitiationReversals())

	t.Run("upsert with unknown connector", func(t *testing.T) {
		connector := models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}
		p := defaultPaymentInitiationReversals()[0]
		p.ID.ConnectorID = connector
		p.ConnectorID = connector

		err := store.PaymentInitiationReversalsUpsert(ctx, p, nil)
		require.Error(t, err)
	})

	t.Run("upsert with same id", func(t *testing.T) {
		pi := models.PaymentInitiationReversal{
			ID:                  pirID1,
			ConnectorID:         defaultConnector.ID,
			PaymentInitiationID: piID1,
			Reference:           "test_changed",
			CreatedAt:           now.Add(-30 * time.Minute).UTC().Time,
			Description:         "test_changed",
			Amount:              big.NewInt(100),
			Asset:               "DKK/2",
			Metadata:            map[string]string{},
		}

		upsertPaymentInitiationReversals(t, ctx, store, []models.PaymentInitiationReversal{pi})

		actual, err := store.PaymentInitiationReversalsGet(ctx, pirID1)
		require.NoError(t, err)
		comparePaymentInitiationReversals(t, defaultPaymentInitiationReversals()[0], *actual)
	})
}

func TestPaymentInitiationReversalsGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())
	upsertPaymentInitiationReversals(t, ctx, store, defaultPaymentInitiationReversals())

	t.Run("get unknown payment initiation reversal", func(t *testing.T) {
		_, err := store.PaymentInitiationReversalsGet(ctx, models.PaymentInitiationReversalID{
			Reference:   "unknown",
			ConnectorID: defaultConnector.ID,
		})
		require.Error(t, err)
		require.ErrorIs(t, err, postgres.ErrNotFound)
	})

	t.Run("get existing payment initiation", func(t *testing.T) {
		for _, pi := range defaultPaymentInitiationReversals() {
			actual, err := store.PaymentInitiationReversalsGet(ctx, pi.ID)
			require.NoError(t, err)
			comparePaymentInitiationReversals(t, pi, *actual)
		}
	})
}

func TestPaymentInitiationReversalsDeleteFromConnectorID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())
	upsertPaymentInitiationReversals(t, ctx, store, defaultPaymentInitiationReversals())

	t.Run("delete from unknown connector", func(t *testing.T) {
		require.NoError(t, store.PaymentInitiationReversalsDeleteFromConnectorID(ctx, models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}))

		for _, pi := range defaultPaymentInitiationReversals() {
			actual, err := store.PaymentInitiationReversalsGet(ctx, pi.ID)
			require.NoError(t, err)
			comparePaymentInitiationReversals(t, pi, *actual)
		}
	})

	t.Run("delete from existing connector", func(t *testing.T) {
		require.NoError(t, store.PaymentInitiationReversalsDeleteFromConnectorID(ctx, defaultConnector.ID))

		for _, pi := range defaultPaymentInitiationReversals() {
			_, err := store.PaymentInitiationReversalsGet(ctx, pi.ID)
			require.Error(t, err)
			require.ErrorIs(t, err, postgres.ErrNotFound)
		}
	})
}

func TestPaymentInitiationReversalsList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())
	upsertPaymentInitiationReversals(t, ctx, store, defaultPaymentInitiationReversals())

	t.Run("wrong query builder when listing by reference", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("reference", "test1")),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
		assert.True(t, errors.Is(err, ErrValidation))
		assert.Regexp(t, "reference", err.Error())
	})

	t.Run("list payment intitiations reversals by id", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", defaultPaymentInitiationReversals()[0].ID.String())),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePaymentInitiationReversals(t, defaultPaymentInitiationReversals()[0], cursor.Data[0])
	})

	t.Run("list payment initiations reversals by unknown id", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", "unknown")),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payment intitiations reversals by reference", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("reference", "test1")),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePaymentInitiationReversals(t, defaultPaymentInitiationReversals()[0], cursor.Data[0])
	})

	t.Run("list payment initiations reversals by unknown reference", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("reference", "unknown")),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payment initiations reversals by connector_id", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connector_id", defaultConnector.ID.String())),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 3)
		require.False(t, cursor.HasMore)
		comparePaymentInitiationReversals(t, defaultPaymentInitiationReversals()[1], cursor.Data[0])
		comparePaymentInitiationReversals(t, defaultPaymentInitiationReversals()[2], cursor.Data[1])
		comparePaymentInitiationReversals(t, defaultPaymentInitiationReversals()[0], cursor.Data[2])
	})

	t.Run("list payment initiations reversals by unknown connector_id", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connector_id", "unknown")),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payment initiations reversals by asset", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("asset", "EUR/2")),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		comparePaymentInitiationReversals(t, defaultPaymentInitiationReversals()[2], cursor.Data[0])
		comparePaymentInitiationReversals(t, defaultPaymentInitiationReversals()[0], cursor.Data[1])
	})

	t.Run("list payment initiations reversals by asset 2", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("asset", "USD/2")),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePaymentInitiationReversals(t, defaultPaymentInitiationReversals()[1], cursor.Data[0])
	})

	t.Run("list payment initiations reversals by unknown asset", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("asset", "unknown")),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payment initiations reversals by payment initiation id", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("payment_initiation_id", piID1)),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		comparePaymentInitiationReversals(t, defaultPaymentInitiationReversals()[1], cursor.Data[0])
		comparePaymentInitiationReversals(t, defaultPaymentInitiationReversals()[0], cursor.Data[1])
	})

	t.Run("list payment initiations reversals by unknowns payment initiation id", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("payment_initiation_id", "unknown")),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payment initiations reversals by amount", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("amount", 200)),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePaymentInitiationReversals(t, defaultPaymentInitiationReversals()[2], cursor.Data[0])
	})

	t.Run("list payment initiations reversals by unknown amount", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("amount", 0)),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("wrong query builder when listing by metadata", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("metadata[foo]", "bar")),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list payment initiations reversals by metadata", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[foo]", "bar")),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePaymentInitiationReversals(t, defaultPaymentInitiationReversals()[1], cursor.Data[0])
	})

	t.Run("list payment initiations reversals by unknown metadata", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[foo]", "unknown")),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payment initiations reversals by unknown metadata 2", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[unknown]", "bar")),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("unknown query builder key when listing", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithQueryBuilder(query.Match("unknown", "bar")),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list payment initiations reversals test cursor", func(t *testing.T) {
		q := NewListPaymentInitiationReversalsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalQuery{}).
				WithPageSize(1),
		)

		cursor, err := store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePaymentInitiationReversals(t, defaultPaymentInitiationReversals()[1], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePaymentInitiationReversals(t, defaultPaymentInitiationReversals()[2], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		comparePaymentInitiationReversals(t, defaultPaymentInitiationReversals()[0], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePaymentInitiationReversals(t, defaultPaymentInitiationReversals()[2], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationReversalsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePaymentInitiationReversals(t, defaultPaymentInitiationReversals()[1], cursor.Data[0])
	})
}

var (
	pirAdjID1 = models.PaymentInitiationReversalAdjustmentID{
		PaymentInitiationReversalID: defaultPaymentInitiationReversals()[0].ID,
		CreatedAt:                   now.Add(-10 * time.Minute).UTC().Time,
		Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSING,
	}
	pirAdjID2 = models.PaymentInitiationReversalAdjustmentID{
		PaymentInitiationReversalID: defaultPaymentInitiationReversals()[0].ID,
		CreatedAt:                   now.Add(-5 * time.Minute).UTC().Time,
		Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_FAILED,
	}
	pirAdjID3 = models.PaymentInitiationReversalAdjustmentID{
		PaymentInitiationReversalID: defaultPaymentInitiationReversals()[1].ID,
		CreatedAt:                   now.Add(-7 * time.Minute).UTC().Time,
		Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED,
	}

	defaultPaymentInitiationReversalAdjustments = []models.PaymentInitiationReversalAdjustment{
		{
			ID:                          pirAdjID1,
			PaymentInitiationReversalID: defaultPaymentInitiationReversals()[0].ID,
			CreatedAt:                   now.Add(-10 * time.Minute).UTC().Time,
			Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSING,
			Metadata: map[string]string{
				"foo": "bar",
			},
		},
		{
			ID:                          pirAdjID2,
			PaymentInitiationReversalID: defaultPaymentInitiationReversals()[0].ID,
			CreatedAt:                   now.Add(-5 * time.Minute).UTC().Time,
			Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_FAILED,
			Error:                       errors.New("test"),
			Metadata: map[string]string{
				"foo2": "bar2",
			},
		},
		{
			ID:                          pirAdjID3,
			PaymentInitiationReversalID: defaultPaymentInitiationReversals()[1].ID,
			CreatedAt:                   now.Add(-7 * time.Minute).UTC().Time,
			Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED,
			Metadata: map[string]string{
				"foo3": "bar3",
			},
		},
	}
)

func upsertPaymentInitiationReversalAdjustments(t *testing.T, ctx context.Context, storage Storage, adjustments []models.PaymentInitiationReversalAdjustment) {
	for _, adj := range adjustments {
		require.NoError(t, storage.PaymentInitiationReversalAdjustmentsUpsert(ctx, adj))
	}
}

func TestPaymentInitiationReversalAdjustmentsUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())
	upsertPaymentInitiationReversals(t, ctx, store, defaultPaymentInitiationReversals())
	upsertPaymentInitiationReversalAdjustments(t, ctx, store, defaultPaymentInitiationReversalAdjustments)

	t.Run("upsert with unknown payment initiation reversal", func(t *testing.T) {
		p := models.PaymentInitiationReversalAdjustment{
			ID:                          models.PaymentInitiationReversalAdjustmentID{},
			PaymentInitiationReversalID: models.PaymentInitiationReversalID{},
			CreatedAt:                   now.Add(-10 * time.Minute).UTC().Time,
			Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSING,
			Metadata: map[string]string{
				"foo": "bar",
			},
		}

		require.Error(t, store.PaymentInitiationReversalAdjustmentsUpsert(ctx, p))
	})

	t.Run("upsert with same id", func(t *testing.T) {
		p := models.PaymentInitiationReversalAdjustment{
			ID:                          pirAdjID1,
			PaymentInitiationReversalID: defaultPaymentInitiationReversalAdjustments[0].PaymentInitiationReversalID,
			CreatedAt:                   now.Add(-30 * time.Minute).UTC().Time,
			Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED,
			Metadata: map[string]string{
				"foo": "changed",
			},
		}

		require.NoError(t, store.PaymentInitiationReversalAdjustmentsUpsert(ctx, p))

		for _, pa := range defaultPaymentInitiationReversalAdjustments {
			actual, err := store.PaymentInitiationReversalAdjustmentsGet(ctx, pa.ID)
			require.NoError(t, err)
			comparePaymentInitiationReversalAdjustments(t, pa, *actual)
		}
	})
}

func TestPaymentInitiationReversalAdjustmentsGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())
	upsertPaymentInitiationReversals(t, ctx, store, defaultPaymentInitiationReversals())
	upsertPaymentInitiationReversalAdjustments(t, ctx, store, defaultPaymentInitiationReversalAdjustments)

	t.Run("get unknown payment initiation adjustment", func(t *testing.T) {
		_, err := store.PaymentInitiationReversalAdjustmentsGet(ctx, models.PaymentInitiationReversalAdjustmentID{})
		require.Error(t, err)
		require.ErrorIs(t, err, postgres.ErrNotFound)
	})

	t.Run("get existing payment initiation adjustment", func(t *testing.T) {
		for _, pa := range defaultPaymentInitiationReversalAdjustments {
			actual, err := store.PaymentInitiationReversalAdjustmentsGet(ctx, pa.ID)
			require.NoError(t, err)
			comparePaymentInitiationReversalAdjustments(t, pa, *actual)
		}
	})
}

func TestPaymentInitiationReversalAdjustmentsList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())
	upsertPaymentInitiations(t, ctx, store, defaultPaymentInitiations())
	upsertPaymentInitiationReversals(t, ctx, store, defaultPaymentInitiationReversals())
	upsertPaymentInitiationReversalAdjustments(t, ctx, store, defaultPaymentInitiationReversalAdjustments)

	t.Run("list payment initiation reversal adjustments by unknown payment initiation", func(t *testing.T) {
		cursor, err := store.PaymentInitiationReversalAdjustmentsList(
			ctx,
			models.PaymentInitiationReversalID{},
			NewListPaymentInitiationReversalAdjustmentsQuery(
				bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalAdjustmentsQuery{}).
					WithPageSize(15),
			),
		)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payment initiation reversal adjustments by payment initiation", func(t *testing.T) {
		q := NewListPaymentInitiationReversalAdjustmentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentInitiationReversalAdjustmentsQuery{}).
				WithPageSize(1),
		)

		cursor, err := store.PaymentInitiationReversalAdjustmentsList(ctx, defaultPaymentInitiationReversalAdjustments[0].PaymentInitiationReversalID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePaymentInitiationReversalAdjustments(t, defaultPaymentInitiationReversalAdjustments[1], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationReversalAdjustmentsList(ctx, defaultPaymentInitiationReversalAdjustments[0].PaymentInitiationReversalID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		comparePaymentInitiationReversalAdjustments(t, defaultPaymentInitiationReversalAdjustments[0], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentInitiationReversalAdjustmentsList(ctx, defaultPaymentInitiationReversalAdjustments[0].PaymentInitiationReversalID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePaymentInitiationReversalAdjustments(t, defaultPaymentInitiationReversalAdjustments[1], cursor.Data[0])
	})
}

func comparePaymentInitiationReversals(t *testing.T, expected, actual models.PaymentInitiationReversal) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.ConnectorID, actual.ConnectorID)
	require.Equal(t, expected.PaymentInitiationID, actual.PaymentInitiationID)
	require.Equal(t, expected.Reference, actual.Reference)
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
	require.Equal(t, expected.Description, actual.Description)
	require.Equal(t, expected.Amount, actual.Amount)
	require.Equal(t, expected.Asset, actual.Asset)

	require.Equal(t, len(expected.Metadata), len(actual.Metadata))
	for k, v := range expected.Metadata {
		_, ok := actual.Metadata[k]
		require.True(t, ok)
		require.Equal(t, v, actual.Metadata[k])
	}
}

func comparePaymentInitiationReversalAdjustments(t *testing.T, expected, actual models.PaymentInitiationReversalAdjustment) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.PaymentInitiationReversalID, actual.PaymentInitiationReversalID)
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
