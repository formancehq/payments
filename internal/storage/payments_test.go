package storage

import (
	"context"
	"math/big"
	"testing"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	pID1 = models.PaymentID{
		PaymentReference: models.PaymentReference{
			Reference: "test1",
			Type:      models.PAYMENT_TYPE_TRANSFER,
		},
		ConnectorID: defaultConnector.ID,
	}

	pid2 = models.PaymentID{
		PaymentReference: models.PaymentReference{
			Reference: "test2",
			Type:      models.PAYMENT_TYPE_PAYIN,
		},
		ConnectorID: defaultConnector.ID,
	}

	pid3 = models.PaymentID{
		PaymentReference: models.PaymentReference{
			Reference: "test3",
			Type:      models.PAYMENT_TYPE_PAYOUT,
		},
		ConnectorID: defaultConnector.ID,
	}
)

func defaultPaymentsRefunded() []models.Payment {
	defaultAccounts := defaultAccounts()
	return []models.Payment{
		{
			ID:                   pID1,
			ConnectorID:          defaultConnector.ID,
			Reference:            "test1",
			CreatedAt:            now.Add(-60 * time.Minute).UTC().Time,
			Type:                 models.PAYMENT_TYPE_TRANSFER,
			InitialAmount:        big.NewInt(100),
			Amount:               big.NewInt(100),
			Asset:                "USD/2",
			Scheme:               models.PAYMENT_SCHEME_OTHER,
			SourceAccountID:      &defaultAccounts[0].ID,
			DestinationAccountID: &defaultAccounts[1].ID,
			Metadata: map[string]string{
				"key1": "value1",
			},
			Adjustments: []models.PaymentAdjustment{
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pID1,
						Reference: "test1",
						CreatedAt: now.Add(-60 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_SUCCEEDED,
					},
					Reference: "test1",
					CreatedAt: now.Add(-60 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_SUCCEEDED,
					Amount:    big.NewInt(100),
					Asset:     pointer.For("USD/2"),
					Raw:       []byte(`{}`),
				},
			},
		},
		{
			ID:                   pID1,
			ConnectorID:          defaultConnector.ID,
			Reference:            "test1",
			CreatedAt:            now.Add(-59 * time.Minute).UTC().Time,
			Type:                 models.PAYMENT_TYPE_TRANSFER,
			InitialAmount:        big.NewInt(100),
			Amount:               big.NewInt(100),
			Asset:                "USD/2",
			Scheme:               models.PAYMENT_SCHEME_OTHER,
			SourceAccountID:      &defaultAccounts[0].ID,
			DestinationAccountID: &defaultAccounts[1].ID,
			Metadata: map[string]string{
				"key1": "value1",
			},
			Adjustments: []models.PaymentAdjustment{
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pID1,
						Reference: "test1",
						CreatedAt: now.Add(-59 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_REFUNDED,
					},
					Reference: "test1",
					CreatedAt: now.Add(-59 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_REFUNDED,
					Amount:    big.NewInt(10),
					Asset:     pointer.For("USD/2"),
					Raw:       []byte(`{}`),
				},
			},
		},
		{
			ID:                   pID1,
			ConnectorID:          defaultConnector.ID,
			Reference:            "test1",
			CreatedAt:            now.Add(-58 * time.Minute).UTC().Time,
			Type:                 models.PAYMENT_TYPE_TRANSFER,
			InitialAmount:        big.NewInt(100),
			Amount:               big.NewInt(100),
			Asset:                "USD/2",
			Scheme:               models.PAYMENT_SCHEME_OTHER,
			SourceAccountID:      &defaultAccounts[0].ID,
			DestinationAccountID: &defaultAccounts[1].ID,
			Metadata: map[string]string{
				"key1": "value1",
			},
			Adjustments: []models.PaymentAdjustment{
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pID1,
						Reference: "test1",
						CreatedAt: now.Add(-58 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_REFUNDED,
					},
					Reference: "test1",
					CreatedAt: now.Add(-58 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_REFUNDED,
					Amount:    big.NewInt(10),
					Asset:     pointer.For("USD/2"),
					Raw:       []byte(`{}`),
				},
			},
		},
	}
}

func defaultPayments() []models.Payment {
	defaultAccounts := defaultAccounts()
	return []models.Payment{
		{
			ID:                   pID1,
			ConnectorID:          defaultConnector.ID,
			Reference:            "test1",
			CreatedAt:            now.Add(-60 * time.Minute).UTC().Time,
			Type:                 models.PAYMENT_TYPE_TRANSFER,
			InitialAmount:        big.NewInt(100),
			Amount:               big.NewInt(100),
			Asset:                "USD/2",
			Scheme:               models.PAYMENT_SCHEME_OTHER,
			SourceAccountID:      &defaultAccounts[0].ID,
			DestinationAccountID: &defaultAccounts[1].ID,
			Metadata: map[string]string{
				"key1": "value1",
			},
			Adjustments: []models.PaymentAdjustment{
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pID1,
						Reference: "test1",
						CreatedAt: now.Add(-60 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_SUCCEEDED,
					},
					Reference: "test1",
					CreatedAt: now.Add(-60 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_SUCCEEDED,
					Amount:    big.NewInt(100),
					Asset:     pointer.For("USD/2"),
					Raw:       []byte(`{}`),
				},
			},
		},
		{
			ID:                   pid2,
			ConnectorID:          defaultConnector.ID,
			Reference:            "test2",
			CreatedAt:            now.Add(-30 * time.Minute).UTC().Time,
			Type:                 models.PAYMENT_TYPE_PAYIN,
			InitialAmount:        big.NewInt(200),
			Amount:               big.NewInt(200),
			Asset:                "EUR/2",
			Scheme:               models.PAYMENT_SCHEME_OTHER,
			DestinationAccountID: &defaultAccounts[0].ID,
			Adjustments: []models.PaymentAdjustment{
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pid2,
						Reference: "test2",
						CreatedAt: now.Add(-30 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_FAILED,
					},
					Reference: "test2",
					CreatedAt: now.Add(-30 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_FAILED,
					Amount:    big.NewInt(200),
					Asset:     pointer.For("EUR/2"),
					Raw:       []byte(`{}`),
				},
			},
		},
		{
			ID:              pid3,
			ConnectorID:     defaultConnector.ID,
			Reference:       "test3",
			CreatedAt:       now.Add(-55 * time.Minute).UTC().Time,
			Type:            models.PAYMENT_TYPE_PAYOUT,
			InitialAmount:   big.NewInt(300),
			Amount:          big.NewInt(300),
			Asset:           "DKK/2",
			Scheme:          models.PAYMENT_SCHEME_A2A,
			SourceAccountID: &defaultAccounts[1].ID,
			Adjustments: []models.PaymentAdjustment{
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pid3,
						Reference: "another_reference",
						CreatedAt: now.Add(-55 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_PENDING,
					},
					Reference: "another_reference",
					CreatedAt: now.Add(-55 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_PENDING,
					Amount:    big.NewInt(300),
					Asset:     pointer.For("DKK/2"),
					Raw:       []byte(`{}`),
				},
			},
		},
	}
}

func upsertPayments(t *testing.T, ctx context.Context, storage Storage, payments []models.Payment) {
	require.NoError(t, storage.PaymentsUpsert(ctx, payments))
}

func TestPaymentsUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())

	t.Run("upsert with unknown connector", func(t *testing.T) {
		connector := models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}
		payments := defaultPayments()
		p := payments[0]
		p.ID = models.PaymentID{
			PaymentReference: models.PaymentReference{
				Reference: "test4",
				Type:      models.PAYMENT_TYPE_PAYOUT,
			},
			ConnectorID: connector,
		}
		p.ConnectorID = connector

		err := store.PaymentsUpsert(ctx, []models.Payment{p})
		require.Error(t, err)
	})

	t.Run("upsert with same id", func(t *testing.T) {
		payments := defaultPayments()
		p := payments[2]
		p.Amount = big.NewInt(200)
		p.Scheme = models.PAYMENT_SCHEME_A2A
		upsertPayments(t, ctx, store, []models.Payment{p})

		// should not have changed
		actual, err := store.PaymentsGet(ctx, p.ID)
		require.NoError(t, err)

		comparePayments(t, payments[2], *actual)
	})

	t.Run("upsert with different adjustments", func(t *testing.T) {
		p := models.Payment{
			ID:              pid3,
			ConnectorID:     defaultConnector.ID,
			Reference:       "test3",
			CreatedAt:       now.Add(-55 * time.Minute).UTC().Time,
			Type:            models.PAYMENT_TYPE_PAYOUT,
			InitialAmount:   big.NewInt(300),
			Amount:          big.NewInt(300),
			Asset:           "DKK/2",
			Scheme:          models.PAYMENT_SCHEME_A2A,
			SourceAccountID: &defaultAccounts()[1].ID,
			Adjustments: []models.PaymentAdjustment{
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pid3,
						Reference: "another_reference2",
						CreatedAt: now.Add(-45 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_SUCCEEDED,
					},
					Reference: "another_reference2",
					CreatedAt: now.Add(-45 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_SUCCEEDED,
					Amount:    big.NewInt(300),
					Asset:     pointer.For("DKK/2"),
					Metadata:  map[string]string{},
					Raw:       []byte(`{}`),
				},
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pid3,
						Reference: "another_reference",
						CreatedAt: now.Add(-55 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_PENDING,
					},
					Reference: "another_reference",
					CreatedAt: now.Add(-55 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_PENDING,
					Amount:    big.NewInt(300),
					Asset:     pointer.For("DKK/2"),
					Raw:       []byte(`{}`),
				},
			},
		}

		upsertPayments(t, ctx, store, []models.Payment{p})

		actual, err := store.PaymentsGet(ctx, p.ID)
		require.NoError(t, err)
		comparePayments(t, p, *actual)
	})

	t.Run("upsert with refund", func(t *testing.T) {
		p := models.Payment{
			ID:            pID1,
			ConnectorID:   defaultConnector.ID,
			InitialAmount: big.NewInt(0),
			Amount:        big.NewInt(0),
			Adjustments: []models.PaymentAdjustment{
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pID1,
						Reference: "test1",
						CreatedAt: now.Add(-20 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_REFUNDED,
					},
					Reference: "test1",
					CreatedAt: now.Add(-20 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_REFUNDED,
					Amount:    big.NewInt(50),
					Asset:     pointer.For("USD/2"),
					Raw:       []byte(`{}`),
				},
			},
		}

		upsertPayments(t, ctx, store, []models.Payment{p})

		actual, err := store.PaymentsGet(ctx, p.ID)
		require.NoError(t, err)

		expectedPayments := models.Payment{
			ID:                   pID1,
			ConnectorID:          defaultConnector.ID,
			Reference:            "test1",
			CreatedAt:            now.Add(-60 * time.Minute).UTC().Time,
			Type:                 models.PAYMENT_TYPE_TRANSFER,
			InitialAmount:        big.NewInt(100),
			Amount:               big.NewInt(50),
			Asset:                "USD/2",
			Scheme:               models.PAYMENT_SCHEME_OTHER,
			Status:               models.PAYMENT_STATUS_REFUNDED,
			SourceAccountID:      &defaultAccounts()[0].ID,
			DestinationAccountID: &defaultAccounts()[1].ID,
			Metadata: map[string]string{
				"key1": "value1",
			},
			Adjustments: []models.PaymentAdjustment{
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pID1,
						Reference: "test1",
						CreatedAt: now.Add(-20 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_REFUNDED,
					},
					Reference: "test1",
					CreatedAt: now.Add(-20 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_REFUNDED,
					Amount:    big.NewInt(50),
					Asset:     pointer.For("USD/2"),
					Raw:       []byte(`{}`),
				},
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pID1,
						Reference: "test1",
						CreatedAt: now.Add(-60 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_SUCCEEDED,
					},
					Reference: "test1",
					CreatedAt: now.Add(-60 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_SUCCEEDED,
					Amount:    big.NewInt(100),
					Asset:     pointer.For("USD/2"),
					Raw:       []byte(`{}`),
				},
			},
		}

		comparePayments(t, expectedPayments, *actual)
	})

	t.Run("upsert with reversed refund", func(t *testing.T) {
		p := models.Payment{
			ID:            pID1,
			ConnectorID:   defaultConnector.ID,
			InitialAmount: big.NewInt(0),
			Amount:        big.NewInt(0),
			Adjustments: []models.PaymentAdjustment{
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pID1,
						Reference: "test1",
						CreatedAt: now.Add(-10 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_REFUND_REVERSED,
					},
					Reference: "test1",
					CreatedAt: now.Add(-10 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_REFUND_REVERSED,
					Amount:    big.NewInt(50),
					Asset:     pointer.For("USD/2"),
					Raw:       []byte(`{}`),
				},
			},
		}

		upsertPayments(t, ctx, store, []models.Payment{p})

		actual, err := store.PaymentsGet(ctx, p.ID)
		require.NoError(t, err)

		expectedPayments := models.Payment{
			ID:                   pID1,
			ConnectorID:          defaultConnector.ID,
			Reference:            "test1",
			CreatedAt:            now.Add(-60 * time.Minute).UTC().Time,
			Type:                 models.PAYMENT_TYPE_TRANSFER,
			InitialAmount:        big.NewInt(100),
			Amount:               big.NewInt(100),
			Asset:                "USD/2",
			Scheme:               models.PAYMENT_SCHEME_OTHER,
			Status:               models.PAYMENT_STATUS_REFUNDED,
			SourceAccountID:      &defaultAccounts()[0].ID,
			DestinationAccountID: &defaultAccounts()[1].ID,
			Metadata: map[string]string{
				"key1": "value1",
			},
			Adjustments: []models.PaymentAdjustment{
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pID1,
						Reference: "test1",
						CreatedAt: now.Add(-10 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_REFUND_REVERSED,
					},
					Reference: "test1",
					CreatedAt: now.Add(-10 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_REFUND_REVERSED,
					Amount:    big.NewInt(50),
					Asset:     pointer.For("USD/2"),
					Raw:       []byte(`{}`),
				},
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pID1,
						Reference: "test1",
						CreatedAt: now.Add(-20 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_REFUNDED,
					},
					Reference: "test1",
					CreatedAt: now.Add(-20 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_REFUNDED,
					Amount:    big.NewInt(50),
					Asset:     pointer.For("USD/2"),
					Raw:       []byte(`{}`),
				},
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pID1,
						Reference: "test1",
						CreatedAt: now.Add(-60 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_SUCCEEDED,
					},
					Reference: "test1",
					CreatedAt: now.Add(-60 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_SUCCEEDED,
					Amount:    big.NewInt(100),
					Asset:     pointer.For("USD/2"),
					Raw:       []byte(`{}`),
				},
			},
		}

		comparePayments(t, expectedPayments, *actual)
	})

	t.Run("upsert with amount adjustment", func(t *testing.T) {
		p := models.Payment{
			ID:            pID1,
			ConnectorID:   defaultConnector.ID,
			InitialAmount: big.NewInt(0),
			Amount:        big.NewInt(0),
			Adjustments: []models.PaymentAdjustment{
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pID1,
						Reference: "test1",
						CreatedAt: now.Add(-5 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_AMOUNT_ADJUSTMENT,
					},
					Reference: "test1",
					CreatedAt: now.Add(-5 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_AMOUNT_ADJUSTMENT,
					Amount:    big.NewInt(150),
					Asset:     pointer.For("USD/2"),
					Raw:       []byte(`{}`),
				},
			},
		}

		upsertPayments(t, ctx, store, []models.Payment{p})

		actual, err := store.PaymentsGet(ctx, p.ID)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(150), actual.InitialAmount)
	})

	t.Run("upsert with capture", func(t *testing.T) {
		p := models.Payment{
			ID:            pID1,
			ConnectorID:   defaultConnector.ID,
			InitialAmount: big.NewInt(0),
			Amount:        big.NewInt(0),
			Adjustments: []models.PaymentAdjustment{
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pID1,
						Reference: "test1",
						CreatedAt: now.Add(-3 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_CAPTURE,
					},
					Reference: "test1",
					CreatedAt: now.Add(-3 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_CAPTURE,
					Amount:    big.NewInt(50),
					Asset:     pointer.For("USD/2"),
					Raw:       []byte(`{}`),
				},
			},
		}

		upsertPayments(t, ctx, store, []models.Payment{p})

		actual, err := store.PaymentsGet(ctx, p.ID)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(150), actual.Amount)
	})
}

func TestPaymentsUpsertRefunded(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPaymentsRefunded())

	actual, err := store.PaymentsGet(ctx, pID1)
	require.NoError(t, err)
	// two refunds in the same batch, should be 100 - 10 - 10 = 80
	require.Equal(t, big.NewInt(80), actual.Amount)
}

func TestPaymentsUpdateMetadata(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())

	t.Run("update metadata of unknown payment", func(t *testing.T) {
		require.Error(t, store.PaymentsUpdateMetadata(ctx, models.PaymentID{
			PaymentReference: models.PaymentReference{Reference: "unknown", Type: models.PAYMENT_TYPE_TRANSFER},
			ConnectorID:      defaultConnector.ID,
		}, map[string]string{}))
	})

	t.Run("update existing metadata", func(t *testing.T) {
		metadata := map[string]string{
			"key1": "changed",
		}
		payments := defaultPayments()
		require.NoError(t, store.PaymentsUpdateMetadata(ctx, payments[0].ID, metadata))

		actual, err := store.PaymentsGet(ctx, payments[0].ID)
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

		payments := defaultPayments()
		require.NoError(t, store.PaymentsUpdateMetadata(ctx, payments[1].ID, metadata))

		actual, err := store.PaymentsGet(ctx, payments[1].ID)
		require.NoError(t, err)
		require.Equal(t, len(metadata), len(actual.Metadata))
		for k, v := range metadata {
			_, ok := actual.Metadata[k]
			require.True(t, ok)
			require.Equal(t, v, actual.Metadata[k])
		}
	})
}

func TestPaymentsGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())

	t.Run("get unknown payment", func(t *testing.T) {
		_, err := store.PaymentsGet(ctx, models.PaymentID{
			PaymentReference: models.PaymentReference{Reference: "unknown", Type: models.PAYMENT_TYPE_TRANSFER},
			ConnectorID:      defaultConnector.ID,
		})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("get existing payments", func(t *testing.T) {
		for _, p := range defaultPayments() {
			actual, err := store.PaymentsGet(ctx, p.ID)
			require.NoError(t, err)
			comparePayments(t, p, *actual)
		}
	})
}

func TestPaymentsGetMultipleAdjustmentsLastStatus(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())

	p := models.Payment{
		ID:                   pID1,
		ConnectorID:          defaultConnector.ID,
		Reference:            "test1",
		CreatedAt:            now.Add(-60 * time.Minute).UTC().Time,
		Type:                 models.PAYMENT_TYPE_TRANSFER,
		InitialAmount:        big.NewInt(100),
		Amount:               big.NewInt(100),
		Asset:                "USD/2",
		Scheme:               models.PAYMENT_SCHEME_OTHER,
		SourceAccountID:      &defaultAccounts()[0].ID,
		DestinationAccountID: &defaultAccounts()[1].ID,
		Metadata: map[string]string{
			"key1": "value1",
		},
		Adjustments: []models.PaymentAdjustment{
			{
				ID: models.PaymentAdjustmentID{
					PaymentID: pID1,
					Reference: "test1",
					CreatedAt: now.Add(-55 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_CAPTURE,
				},
				Reference: "test1",
				CreatedAt: now.Add(-55 * time.Minute).UTC().Time,
				Status:    models.PAYMENT_STATUS_CAPTURE,
				Amount:    big.NewInt(100),
				Asset:     pointer.For("USD/2"),
				Raw:       []byte(`{}`),
			},
			{
				ID: models.PaymentAdjustmentID{
					PaymentID: pID1,
					Reference: "test1",
					CreatedAt: now.Add(-60 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_AUTHORISATION,
				},
				Reference: "test1",
				CreatedAt: now.Add(-60 * time.Minute).UTC().Time,
				Status:    models.PAYMENT_STATUS_AUTHORISATION,
				Amount:    big.NewInt(100),
				Asset:     pointer.For("USD/2"),
				Raw:       []byte(`{}`),
			},
		},
	}

	upsertPayments(t, ctx, store, []models.Payment{p})

	actual, err := store.PaymentsGet(ctx, p.ID)
	require.NoError(t, err)
	require.Len(t, actual.Adjustments, 2)
	require.Equal(t, models.PAYMENT_STATUS_CAPTURE, actual.Status)
}

func TestPaymentsDeleteFromConnectorID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())

	t.Run("delete from unknown connector", func(t *testing.T) {
		require.NoError(t, store.PaymentsDeleteFromConnectorID(ctx, models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}))

		for _, p := range defaultPayments() {
			actual, err := store.PaymentsGet(ctx, p.ID)
			require.NoError(t, err)
			comparePayments(t, p, *actual)
		}
	})

	t.Run("delete from existing connector", func(t *testing.T) {
		require.NoError(t, store.PaymentsDeleteFromConnectorID(ctx, defaultConnector.ID))

		for _, p := range defaultPayments() {
			_, err := store.PaymentsGet(ctx, p.ID)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrNotFound)
		}
	})
}

func TestPaymentsListSorting(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())

	p := models.Payment{
		ID:                   pID1,
		ConnectorID:          defaultConnector.ID,
		Reference:            "test1",
		CreatedAt:            now.Add(-60 * time.Minute).UTC().Time,
		Type:                 models.PAYMENT_TYPE_TRANSFER,
		InitialAmount:        big.NewInt(100),
		Amount:               big.NewInt(100),
		Asset:                "USD/2",
		Scheme:               models.PAYMENT_SCHEME_OTHER,
		SourceAccountID:      &defaultAccounts()[0].ID,
		DestinationAccountID: &defaultAccounts()[1].ID,
		Metadata: map[string]string{
			"key1": "value1",
		},
		Adjustments: []models.PaymentAdjustment{
			{
				ID: models.PaymentAdjustmentID{
					PaymentID: pID1,
					Reference: "test1",
					CreatedAt: now.Add(-60 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_PENDING,
				},
				Reference: "test1",
				CreatedAt: now.Add(-60 * time.Minute).UTC().Time,
				Status:    models.PAYMENT_STATUS_PENDING,
				Amount:    big.NewInt(100),
				Asset:     pointer.For("USD/2"),
				Raw:       []byte(`{}`),
			},
			{
				ID: models.PaymentAdjustmentID{
					PaymentID: pID1,
					Reference: "test1",
					CreatedAt: now.Add(-60 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_SUCCEEDED,
				},
				Reference: "test1",
				CreatedAt: now.Add(-60 * time.Minute).UTC().Time,
				Status:    models.PAYMENT_STATUS_SUCCEEDED,
				Amount:    big.NewInt(100),
				Asset:     pointer.For("USD/2"),
				Raw:       []byte(`{}`),
			},
		},
	}

	upsertPayments(t, ctx, store, []models.Payment{p})

	q := NewListPaymentsQuery(
		bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
			WithPageSize(1),
	)

	cursor, err := store.PaymentsList(ctx, q)
	require.NoError(t, err)
	require.Len(t, cursor.Data, 1)
	require.False(t, cursor.HasMore)
	require.Empty(t, cursor.Previous)
	require.Empty(t, cursor.Next)

	require.Equal(t, models.PAYMENT_STATUS_SUCCEEDED, cursor.Data[0].Status)
}

func TestPaymentsList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())

	dps := []models.Payment{
		{
			ID:                   pID1,
			ConnectorID:          defaultConnector.ID,
			Reference:            "test1",
			CreatedAt:            now.Add(-60 * time.Minute).UTC().Time,
			Type:                 models.PAYMENT_TYPE_TRANSFER,
			InitialAmount:        big.NewInt(100),
			Amount:               big.NewInt(100),
			Asset:                "USD/2",
			Scheme:               models.PAYMENT_SCHEME_OTHER,
			Status:               models.PAYMENT_STATUS_SUCCEEDED,
			SourceAccountID:      &defaultAccounts()[0].ID,
			DestinationAccountID: &defaultAccounts()[1].ID,
			Metadata: map[string]string{
				"key1": "value1",
			},
		},
		{
			ID:                   pid2,
			ConnectorID:          defaultConnector.ID,
			Reference:            "test2",
			CreatedAt:            now.Add(-30 * time.Minute).UTC().Time,
			Type:                 models.PAYMENT_TYPE_PAYIN,
			InitialAmount:        big.NewInt(200),
			Amount:               big.NewInt(200),
			Asset:                "EUR/2",
			Scheme:               models.PAYMENT_SCHEME_OTHER,
			Status:               models.PAYMENT_STATUS_FAILED,
			DestinationAccountID: &defaultAccounts()[0].ID,
		},
		{
			ID:              pid3,
			ConnectorID:     defaultConnector.ID,
			Reference:       "test3",
			CreatedAt:       now.Add(-55 * time.Minute).UTC().Time,
			Type:            models.PAYMENT_TYPE_PAYOUT,
			InitialAmount:   big.NewInt(300),
			Amount:          big.NewInt(300),
			Asset:           "DKK/2",
			Scheme:          models.PAYMENT_SCHEME_A2A,
			Status:          models.PAYMENT_STATUS_PENDING,
			SourceAccountID: &defaultAccounts()[1].ID,
		},
	}

	t.Run("wrong query builder operator when listing with reference", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("reference", "test1")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
		assert.True(t, errors.Is(err, ErrValidation))
		assert.Regexp(t, "reference", err.Error())
	})

	t.Run("list payments by reference", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("reference", "test1")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePayments(t, dps[0], cursor.Data[0])
	})

	t.Run("list payments by unknown reference", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("reference", "unknown")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payments by id", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", dps[0].ID.String())),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePayments(t, dps[0], cursor.Data[0])
	})

	t.Run("list payments by unknown id", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", "unknown")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payments by connector_id", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connector_id", defaultConnector.ID.String())),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 3)
		require.False(t, cursor.HasMore)
		comparePayments(t, dps[1], cursor.Data[0])
		comparePayments(t, dps[2], cursor.Data[1])
		comparePayments(t, dps[0], cursor.Data[2])
	})

	t.Run("list payments by unknown connector_id", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connector_id", "unknown")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payments by type", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("type", "PAYOUT")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePayments(t, dps[2], cursor.Data[0])
	})

	t.Run("list payments by unknown type", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("type", "UNKNOWN")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payments by asset", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("asset", "EUR/2")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePayments(t, dps[1], cursor.Data[0])
	})

	t.Run("list payments by unknown asset", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("asset", "UNKNOWN")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payments by scheme", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("scheme", "OTHER")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		comparePayments(t, dps[1], cursor.Data[0])
		comparePayments(t, dps[0], cursor.Data[1])
	})

	t.Run("list payments by unknown scheme", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("scheme", "UNKNOWN")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payments by status", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("status", "PENDING")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePayments(t, dps[2], cursor.Data[0])
	})

	t.Run("list payments by unknown status", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("status", "UNKNOWN")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payments by source account id", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("source_account_id", defaultAccounts()[0].ID.String())),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePayments(t, dps[0], cursor.Data[0])
	})

	t.Run("list payments by unknown source account id", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("source_account_id", "unknown")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payments by destination account id", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("destination_account_id", defaultAccounts()[0].ID.String())),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePayments(t, dps[1], cursor.Data[0])
	})

	t.Run("list payments by unknown destination account id", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("destination_account_id", "unknown")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payments by amount", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("amount", 200)),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePayments(t, dps[1], cursor.Data[0])
	})

	t.Run("list payments by unknown amount", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("amount", 0)),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payments by initial_amount", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("initial_amount", 300)),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePayments(t, dps[2], cursor.Data[0])
	})

	t.Run("list payments by unknown initial_amount", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("initial_amount", 0)),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("wrong query builder operator when listing by metadata", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("metadata[key1]", "value1")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list payments by metadata", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[key1]", "value1")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePayments(t, dps[0], cursor.Data[0])
	})

	t.Run("list payments by unknown metadata", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[key1]", "unknown")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("list payments by unknown metadata 2", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[unknown]", "unknown")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
	})

	t.Run("unknown query builder key when listing", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("unknown", "unknown")),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list payments test cursor", func(t *testing.T) {
		q := NewListPaymentsQuery(
			bunpaginate.NewPaginatedQueryOptions(PaymentQuery{}).
				WithPageSize(1),
		)

		cursor, err := store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePayments(t, dps[1], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePayments(t, dps[2], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		comparePayments(t, dps[0], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePayments(t, dps[2], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePayments(t, dps[1], cursor.Data[0])
	})
}

func comparePayments(t *testing.T, expected, actual models.Payment) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.ConnectorID, actual.ConnectorID)
	require.Equal(t, expected.Reference, actual.Reference)
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
	require.Equal(t, expected.Type, actual.Type)
	require.Equal(t, expected.InitialAmount, actual.InitialAmount)
	require.Equal(t, expected.Amount, actual.Amount)
	require.Equal(t, expected.Asset, actual.Asset)
	require.Equal(t, expected.Scheme, actual.Scheme)

	switch {
	case expected.SourceAccountID == nil:
		require.Nil(t, actual.SourceAccountID)
	default:
		require.NotNil(t, actual.SourceAccountID)
		require.Equal(t, *expected.SourceAccountID, *actual.SourceAccountID)
	}

	switch {
	case expected.DestinationAccountID == nil:
		require.Nil(t, actual.DestinationAccountID)
	default:
		require.NotNil(t, actual.DestinationAccountID)
		require.Equal(t, *expected.DestinationAccountID, *actual.DestinationAccountID)
	}

	require.Equal(t, len(expected.Metadata), len(actual.Metadata))
	for k, v := range expected.Metadata {
		_, ok := actual.Metadata[k]
		require.True(t, ok)
		require.Equal(t, v, actual.Metadata[k])
	}

	require.Equal(t, len(expected.Adjustments), len(actual.Adjustments))
	for i := range expected.Adjustments {
		comparePaymentAdjustments(t, expected.Adjustments[i], actual.Adjustments[i])
	}
}

func comparePaymentAdjustments(t *testing.T, expected, actual models.PaymentAdjustment) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
	require.Equal(t, expected.Status, actual.Status)
	require.Equal(t, expected.Amount, actual.Amount)
	require.Equal(t, expected.Asset, actual.Asset)
}

func TestPaymentsDelete(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())

	t.Run("delete existing payment", func(t *testing.T) {
		require.NoError(t, store.PaymentsDelete(ctx, pID1))

		_, err := store.PaymentsGet(ctx, pID1)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("delete non-existing payment", func(t *testing.T) {
		nonExistingID := models.PaymentID{
			PaymentReference: models.PaymentReference{
				Reference: "non-existing",
				Type:      models.PAYMENT_TYPE_TRANSFER,
			},
			ConnectorID: defaultConnector.ID,
		}

		require.NoError(t, store.PaymentsDelete(ctx, nonExistingID))

		// Verify other payments still exist
		payment, err := store.PaymentsGet(ctx, pid2)
		require.NoError(t, err)
		require.NotNil(t, payment)
		require.Equal(t, pid2, payment.ID)
	})
}

func TestPaymentsDeleteFromReference(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())

	t.Run("delete payment by existing reference and connector", func(t *testing.T) {
		require.NoError(t, store.PaymentsDeleteFromReference(ctx, "test1", defaultConnector.ID))

		_, err := store.PaymentsGet(ctx, pID1)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)

		// Verify other payments still exist
		payment, err := store.PaymentsGet(ctx, pid2)
		require.NoError(t, err)
		require.NotNil(t, payment)
		require.Equal(t, pid2, payment.ID)
	})

	t.Run("delete payment by non-existing reference", func(t *testing.T) {
		require.NoError(t, store.PaymentsDeleteFromReference(ctx, "non-existing", defaultConnector.ID))

		// Verify payments still exist
		payment, err := store.PaymentsGet(ctx, pid2)
		require.NoError(t, err)
		require.NotNil(t, payment)
		require.Equal(t, pid2, payment.ID)
	})

	t.Run("delete payment by existing reference but wrong connector", func(t *testing.T) {
		wrongConnectorID := models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "wrong-provider",
		}

		require.NoError(t, store.PaymentsDeleteFromReference(ctx, "test2", wrongConnectorID))

		// Verify payment still exists
		payment, err := store.PaymentsGet(ctx, pid2)
		require.NoError(t, err)
		require.NotNil(t, payment)
		require.Equal(t, pid2, payment.ID)
	})
}

func TestPaymentsDeleteFromAccountID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())

	t.Run("delete payments by source account ID", func(t *testing.T) {
		sourceAccountID := defaultAccounts()[0].ID
		require.NoError(t, store.PaymentsDeleteFromAccountID(ctx, sourceAccountID))

		// Verify payment with source account ID is deleted
		_, err := store.PaymentsGet(ctx, pID1)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)

		// Verify payment with destination account ID is deleted
		_, err = store.PaymentsGet(ctx, pid2)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)

		// Verify payment with different source account ID still exists
		payment, err := store.PaymentsGet(ctx, pid3)
		require.NoError(t, err)
		require.NotNil(t, payment)
		require.Equal(t, pid3, payment.ID)
	})

	t.Run("delete payments by destination account ID", func(t *testing.T) {
		// Re-insert payments for this test
		upsertPayments(t, ctx, store, defaultPayments())

		destinationAccountID := defaultAccounts()[1].ID
		require.NoError(t, store.PaymentsDeleteFromAccountID(ctx, destinationAccountID))

		// Verify payment with destination account ID is deleted
		_, err := store.PaymentsGet(ctx, pID1)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)

		// Verify payment with different destination account ID still exists
		payment, err := store.PaymentsGet(ctx, pid2)
		require.NoError(t, err)
		require.NotNil(t, payment)
		require.Equal(t, pid2, payment.ID)
	})

	t.Run("delete payments by non-existing account ID", func(t *testing.T) {
		// Re-insert payments for this test
		upsertPayments(t, ctx, store, defaultPayments())

		nonExistingAccountID := models.AccountID{
			Reference:   "non-existing",
			ConnectorID: defaultConnector.ID,
		}

		require.NoError(t, store.PaymentsDeleteFromAccountID(ctx, nonExistingAccountID))

		// Verify all payments still exist
		for _, p := range defaultPayments() {
			payment, err := store.PaymentsGet(ctx, p.ID)
			require.NoError(t, err)
			require.NotNil(t, payment)
			require.Equal(t, p.ID, payment.ID)
		}
	})
}
