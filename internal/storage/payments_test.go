package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/query"
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

	// Helper to clean up outbox events created during tests
	cleanupOutbox := func() {
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 1000)
		require.NoError(t, err)
		for _, event := range pendingEvents {
			eventSent := models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: event.IdempotencyKey,
					ConnectorID:         event.ConnectorID,
				},
				ConnectorID: event.ConnectorID,
				SentAt:      time.Now().UTC(),
			}
			_ = store.OutboxEventsDeleteAndRecordSent(ctx, event.ID, eventSent)
		}
	}
	defer cleanupOutbox()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())
	cleanupOutbox() // Clean up outbox events from default data

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

	t.Run("outbox events created for new payment adjustments", func(t *testing.T) {
		accounts := defaultAccounts()
		// Create new payment with adjustment
		newPayment := models.Payment{
			ID: models.PaymentID{
				PaymentReference: models.PaymentReference{
					Reference: "outbox-test-1",
					Type:      models.PAYMENT_TYPE_TRANSFER,
				},
				ConnectorID: defaultConnector.ID,
			},
			ConnectorID:          defaultConnector.ID,
			Reference:            "outbox-test-1",
			CreatedAt:            now.Add(-5 * time.Minute).UTC().Time,
			Type:                 models.PAYMENT_TYPE_TRANSFER,
			InitialAmount:        big.NewInt(500),
			Amount:               big.NewInt(500),
			Asset:                "USD/2",
			Scheme:               models.PAYMENT_SCHEME_OTHER,
			SourceAccountID:      &accounts[0].ID,
			DestinationAccountID: &accounts[1].ID,
			Metadata: map[string]string{
				"test": "outbox",
			},
			Adjustments: []models.PaymentAdjustment{
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: models.PaymentID{
							PaymentReference: models.PaymentReference{
								Reference: "outbox-test-1",
								Type:      models.PAYMENT_TYPE_TRANSFER,
							},
							ConnectorID: defaultConnector.ID,
						},
						Reference: "adj1",
						CreatedAt: now.Add(-5 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_SUCCEEDED,
					},
					Reference: "adj1",
					CreatedAt: now.Add(-5 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_SUCCEEDED,
					Amount:    big.NewInt(500),
					Asset:     pointer.For("USD/2"),
					Raw:       []byte(`{"test": "data"}`),
				},
			},
		}

		// Create a set of expected idempotency keys
		expectedKeys := make(map[string]bool)
		for _, adj := range newPayment.Adjustments {
			expectedKeys[adj.IdempotencyKey()] = true
		}

		// Insert payment
		require.NoError(t, store.PaymentsUpsert(ctx, []models.Payment{newPayment}))

		// Verify outbox events were created
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		// Filter events to only those we just created
		ourEvents := make([]models.OutboxEvent, 0)
		for _, event := range pendingEvents {
			if event.EventType == "payment.saved" && expectedKeys[event.IdempotencyKey] {
				ourEvents = append(ourEvents, event)
			}
		}
		require.Len(t, ourEvents, 1, "expected 1 outbox event for 1 new payment adjustment")

		// Check event details
		event := ourEvents[0]
		assert.Equal(t, "payment.saved", event.EventType)
		assert.Equal(t, models.OUTBOX_STATUS_PENDING, event.Status)
		assert.Equal(t, defaultConnector.ID, *event.ConnectorID)
		assert.Equal(t, 0, event.RetryCount)
		assert.Nil(t, event.Error)
		assert.NotEqual(t, uuid.Nil, event.ID)
		assert.NotEmpty(t, event.IdempotencyKey)

		// Find the matching adjustment by idempotency key
		var expectedAdj models.PaymentAdjustment
		for _, adj := range newPayment.Adjustments {
			if adj.IdempotencyKey() == event.IdempotencyKey {
				expectedAdj = adj
				break
			}
		}

		// Verify payload contains payment data
		var payload map[string]interface{}
		err = json.Unmarshal(event.Payload, &payload)
		require.NoError(t, err)
		assert.Equal(t, newPayment.ID.String(), payload["id"])
		assert.Equal(t, newPayment.Type.String(), payload["type"])
		assert.Equal(t, expectedAdj.Status.String(), payload["status"])
		assert.Equal(t, newPayment.InitialAmount.String(), payload["initialAmount"])
		assert.Equal(t, newPayment.Amount.String(), payload["amount"])
		assert.Equal(t, newPayment.Scheme.String(), payload["scheme"])
		assert.Equal(t, newPayment.Asset, payload["asset"])
		assert.Equal(t, newPayment.ConnectorID.String(), payload["connectorID"])
		assert.Contains(t, payload, "provider")
		assert.Contains(t, payload, "createdAt")
		assert.Equal(t, accounts[0].ID.String(), payload["sourceAccountID"])
		assert.Equal(t, accounts[1].ID.String(), payload["destinationAccountID"])

		// Verify EntityID matches payment ID
		assert.Equal(t, newPayment.ID.String(), event.EntityID)

		// Verify idempotency key matches adjustment
		assert.Equal(t, expectedAdj.IdempotencyKey(), event.IdempotencyKey)
	})

	t.Run("outbox events created for multiple adjustments", func(t *testing.T) {
		accounts := defaultAccounts()
		// Create payment with multiple adjustments
		multiAdjustmentPayment := models.Payment{
			ID: models.PaymentID{
				PaymentReference: models.PaymentReference{
					Reference: "outbox-test-2",
					Type:      models.PAYMENT_TYPE_PAYIN,
				},
				ConnectorID: defaultConnector.ID,
			},
			ConnectorID:          defaultConnector.ID,
			Reference:            "outbox-test-2",
			CreatedAt:            now.Add(-4 * time.Minute).UTC().Time,
			Type:                 models.PAYMENT_TYPE_PAYIN,
			InitialAmount:        big.NewInt(1000),
			Amount:               big.NewInt(1000),
			Asset:                "EUR/2",
			Scheme:               models.PAYMENT_SCHEME_A2A,
			DestinationAccountID: &accounts[0].ID,
			Metadata: map[string]string{
				"test": "multi",
			},
			Adjustments: []models.PaymentAdjustment{
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: models.PaymentID{
							PaymentReference: models.PaymentReference{
								Reference: "outbox-test-2",
								Type:      models.PAYMENT_TYPE_PAYIN,
							},
							ConnectorID: defaultConnector.ID,
						},
						Reference: "adj1",
						CreatedAt: now.Add(-4 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_PENDING,
					},
					Reference: "adj1",
					CreatedAt: now.Add(-4 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_PENDING,
					Amount:    big.NewInt(1000),
					Asset:     pointer.For("EUR/2"),
					Raw:       []byte(`{}`),
				},
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: models.PaymentID{
							PaymentReference: models.PaymentReference{
								Reference: "outbox-test-2",
								Type:      models.PAYMENT_TYPE_PAYIN,
							},
							ConnectorID: defaultConnector.ID,
						},
						Reference: "adj2",
						CreatedAt: now.Add(-3 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_SUCCEEDED,
					},
					Reference: "adj2",
					CreatedAt: now.Add(-3 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_SUCCEEDED,
					Amount:    big.NewInt(1000),
					Asset:     pointer.For("EUR/2"),
					Raw:       []byte(`{}`),
				},
			},
		}

		// Create a set of expected idempotency keys
		expectedKeys := make(map[string]bool)
		for _, adj := range multiAdjustmentPayment.Adjustments {
			expectedKeys[adj.IdempotencyKey()] = true
		}

		// Insert payment
		require.NoError(t, store.PaymentsUpsert(ctx, []models.Payment{multiAdjustmentPayment}))

		// Verify outbox events were created for all adjustments
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		// Filter events to only those we just created
		ourEvents := make([]models.OutboxEvent, 0)
		for _, event := range pendingEvents {
			if event.EventType == "payment.saved" && expectedKeys[event.IdempotencyKey] {
				ourEvents = append(ourEvents, event)
			}
		}
		require.Len(t, ourEvents, 2, "expected 2 outbox events for 2 adjustments")

		// Verify all events have correct structure
		for _, event := range ourEvents {
			assert.Equal(t, "payment.saved", event.EventType)
			assert.Equal(t, models.OUTBOX_STATUS_PENDING, event.Status)
			assert.Equal(t, defaultConnector.ID, *event.ConnectorID)
			assert.NotEqual(t, uuid.Nil, event.ID)
			assert.NotEmpty(t, event.IdempotencyKey)
			assert.Equal(t, multiAdjustmentPayment.ID.String(), event.EntityID)
		}
	})
}

func TestPaymentsUpsertRefunded(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	// Helper to clean up outbox events created during tests
	cleanupOutbox := func() {
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 1000)
		require.NoError(t, err)
		for _, event := range pendingEvents {
			eventSent := models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: event.IdempotencyKey,
					ConnectorID:         event.ConnectorID,
				},
				ConnectorID: event.ConnectorID,
				SentAt:      time.Now().UTC(),
			}
			_ = store.OutboxEventsDeleteAndRecordSent(ctx, event.ID, eventSent)
		}
	}
	defer cleanupOutbox()

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

	switch expected.SourceAccountID {
	case nil:
		require.Nil(t, actual.SourceAccountID)
	default:
		require.NotNil(t, actual.SourceAccountID)
		require.Equal(t, *expected.SourceAccountID, *actual.SourceAccountID)
	}

	switch expected.DestinationAccountID {
	case nil:
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

	// Helper to clean up outbox events created during tests
	cleanupOutbox := func() {
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 1000)
		require.NoError(t, err)
		for _, event := range pendingEvents {
			eventSent := models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: event.IdempotencyKey,
					ConnectorID:         event.ConnectorID,
				},
				ConnectorID: event.ConnectorID,
				SentAt:      time.Now().UTC(),
			}
			_ = store.OutboxEventsDeleteAndRecordSent(ctx, event.ID, eventSent)
		}
	}
	defer cleanupOutbox()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertPayments(t, ctx, store, defaultPayments())
	cleanupOutbox() // Clean up outbox events from account creation

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

	t.Run("outbox events created for deleted payments", func(t *testing.T) {
		// Create a new payment to delete
		accounts := defaultAccounts()
		deletePaymentID := models.PaymentID{
			PaymentReference: models.PaymentReference{
				Reference: "delete-test-1",
				Type:      models.PAYMENT_TYPE_PAYOUT,
			},
			ConnectorID: defaultConnector.ID,
		}

		deletePayment := models.Payment{
			ID:              deletePaymentID,
			ConnectorID:     defaultConnector.ID,
			Reference:       "delete-test-1",
			CreatedAt:       now.Add(-2 * time.Minute).UTC().Time,
			Type:            models.PAYMENT_TYPE_PAYOUT,
			InitialAmount:   big.NewInt(250),
			Amount:          big.NewInt(250),
			Asset:           "GBP/2",
			Scheme:          models.PAYMENT_SCHEME_OTHER,
			SourceAccountID: &accounts[0].ID,
			Adjustments: []models.PaymentAdjustment{
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: deletePaymentID,
						Reference: "del-adj1",
						CreatedAt: now.Add(-2 * time.Minute).UTC().Time,
						Status:    models.PAYMENT_STATUS_SUCCEEDED,
					},
					Reference: "del-adj1",
					CreatedAt: now.Add(-2 * time.Minute).UTC().Time,
					Status:    models.PAYMENT_STATUS_SUCCEEDED,
					Amount:    big.NewInt(250),
					Asset:     pointer.For("GBP/2"),
					Raw:       []byte(`{}`),
				},
			},
		}

		// Insert payment first
		require.NoError(t, store.PaymentsUpsert(ctx, []models.Payment{deletePayment}))

		// Clean up payment.saved events from insertion
		cleanupOutbox()

		expectedKey := fmt.Sprintf("delete:%s", deletePaymentID.String())

		// Delete the payment
		require.NoError(t, store.PaymentsDeleteFromReference(ctx, "delete-test-1", defaultConnector.ID))

		// Verify outbox event was created
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		// Filter events to only the one we just created
		var ourEvent *models.OutboxEvent
		for _, event := range pendingEvents {
			if event.EventType == "payment.deleted" && event.IdempotencyKey == expectedKey {
				ourEvent = &event
				break
			}
		}
		require.NotNil(t, ourEvent, "expected 1 outbox event for deleted payment")

		// Check event details
		assert.Equal(t, "payment.deleted", ourEvent.EventType)
		assert.Equal(t, models.OUTBOX_STATUS_PENDING, ourEvent.Status)
		assert.Equal(t, defaultConnector.ID, *ourEvent.ConnectorID)
		assert.Equal(t, 0, ourEvent.RetryCount)
		assert.Nil(t, ourEvent.Error)
		assert.NotEqual(t, uuid.Nil, ourEvent.ID)
		assert.Equal(t, expectedKey, ourEvent.IdempotencyKey)

		// Verify payload contains payment ID
		var payload map[string]interface{}
		err = json.Unmarshal(ourEvent.Payload, &payload)
		require.NoError(t, err)
		assert.Equal(t, deletePaymentID.String(), payload["id"])

		// Verify EntityID matches payment ID
		assert.Equal(t, deletePaymentID.String(), ourEvent.EntityID)
	})

	t.Run("no outbox events when deleting non-existent payment", func(t *testing.T) {
		// Get count of payment.deleted events before deletion
		allEventsBefore, err := store.OutboxEventsPollPending(ctx, 1000)
		require.NoError(t, err)
		deletedEventsBefore := 0
		for _, event := range allEventsBefore {
			if event.EventType == "payment.deleted" {
				deletedEventsBefore++
			}
		}

		// Try to delete non-existent payment
		require.NoError(t, store.PaymentsDeleteFromReference(ctx, "non-existent", defaultConnector.ID))

		// Get count of payment.deleted events after deletion
		allEventsAfter, err := store.OutboxEventsPollPending(ctx, 1000)
		require.NoError(t, err)
		deletedEventsAfter := 0
		for _, event := range allEventsAfter {
			if event.EventType == "payment.deleted" {
				deletedEventsAfter++
			}
		}

		// Verify no new deleted events were created
		assert.Equal(t, deletedEventsBefore, deletedEventsAfter, "deleting non-existent payment should not create outbox event")
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
