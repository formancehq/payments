package models_test

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromPSPPaymentToPayment(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	t.Run("parent reference is empty", func(t *testing.T) {
		t.Parallel()
		// Given

		pspPayment := models.PSPPayment{
			ParentReference:        "",
			Reference:              "test1",
			CreatedAt:              now.UTC(),
			Type:                   models.PAYMENT_TYPE_PAYOUT,
			Amount:                 big.NewInt(100),
			Asset:                  "USD/2",
			Scheme:                 models.PAYMENT_SCHEME_OTHER,
			Status:                 models.PAYMENT_STATUS_CANCELLED,
			SourceAccountReference: pointer.For("acc"),
			Metadata: map[string]string{
				"foo": "bar",
			},
			Raw: []byte(`{}`),
		}

		actual, err := models.FromPSPPaymentToPayment(pspPayment, connectorID)
		
		// Then
		require.NoError(t, err)

		pid := models.PaymentID{
			PaymentReference: models.PaymentReference{
				Reference: "test1",
				Type:      models.PAYMENT_TYPE_PAYOUT,
			},
			ConnectorID: connectorID,
		}
		expected := models.Payment{
			ID:            pid,
			ConnectorID:   connectorID,
			Reference:     "test1",
			CreatedAt:     now.UTC(),
			Type:          models.PAYMENT_TYPE_PAYOUT,
			InitialAmount: big.NewInt(100),
			Amount:        big.NewInt(100),
			Asset:         "USD/2",
			Scheme:        models.PAYMENT_SCHEME_OTHER,
			Status:        models.PAYMENT_STATUS_CANCELLED,
			SourceAccountID: &models.AccountID{
				Reference:   "acc",
				ConnectorID: connectorID,
			},
			Metadata: map[string]string{
				"foo": "bar",
			},
			Adjustments: []models.PaymentAdjustment{
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pid,
						Reference: "test1",
						CreatedAt: now.UTC(),
						Status:    models.PAYMENT_STATUS_CANCELLED,
					},
					Reference: "test1",
					CreatedAt: now.UTC(),
					Status:    models.PAYMENT_STATUS_CANCELLED,
					Amount:    big.NewInt(100),
					Asset:     pointer.For("USD/2"),
					Metadata: map[string]string{
						"foo": "bar",
					},
					Raw: []byte(`{}`),
				},
			},
		}

		comparePayment(t, expected, actual)
	})

	t.Run("parent reference is not empty", func(t *testing.T) {
		t.Parallel()
		// Given

		pspPayment := models.PSPPayment{
			ParentReference:             "parent_reference",
			Reference:                   "test1",
			CreatedAt:                   now.UTC(),
			Type:                        models.PAYMENT_TYPE_TRANSFER,
			Amount:                      big.NewInt(150),
			Asset:                       "EUR/2",
			Scheme:                      models.PAYMENT_SCHEME_OTHER,
			Status:                      models.PAYMENT_STATUS_SUCCEEDED,
			DestinationAccountReference: pointer.For("acc"),
			Metadata: map[string]string{
				"foo": "bar",
			},
			Raw: []byte(`{}`),
		}

		actual, err := models.FromPSPPaymentToPayment(pspPayment, connectorID)
		
		// Then
		require.NoError(t, err)

		pid := models.PaymentID{
			PaymentReference: models.PaymentReference{
				Reference: "parent_reference",
				Type:      models.PAYMENT_TYPE_TRANSFER,
			},
			ConnectorID: connectorID,
		}
		expected := models.Payment{
			ID:            pid,
			ConnectorID:   connectorID,
			Reference:     "parent_reference",
			CreatedAt:     now.UTC(),
			Type:          models.PAYMENT_TYPE_TRANSFER,
			InitialAmount: big.NewInt(150),
			Amount:        big.NewInt(150),
			Asset:         "EUR/2",
			Scheme:        models.PAYMENT_SCHEME_OTHER,
			Status:        models.PAYMENT_STATUS_SUCCEEDED,
			DestinationAccountID: &models.AccountID{
				Reference:   "acc",
				ConnectorID: connectorID,
			},
			Metadata: map[string]string{
				"foo": "bar",
			},
			Adjustments: []models.PaymentAdjustment{
				{
					ID: models.PaymentAdjustmentID{
						PaymentID: pid,
						Reference: "test1",
						CreatedAt: now.UTC(),
						Status:    models.PAYMENT_STATUS_SUCCEEDED,
					},
					Reference: "test1",
					CreatedAt: now.UTC(),
					Status:    models.PAYMENT_STATUS_SUCCEEDED,
					Amount:    big.NewInt(150),
					Asset:     pointer.For("EUR/2"),
					Metadata: map[string]string{
						"foo": "bar",
					},
					Raw: []byte(`{}`),
				},
			},
		}

		comparePayment(t, expected, actual)
	})
}

func comparePayment(t *testing.T, expected, actual models.Payment) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.ConnectorID, actual.ConnectorID)
	require.Equal(t, expected.Reference, actual.Reference)
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
	require.Equal(t, expected.Type, actual.Type)
	require.Equal(t, expected.InitialAmount, actual.InitialAmount)
	require.Equal(t, expected.Amount, actual.Amount)
	require.Equal(t, expected.Asset, actual.Asset)
	require.Equal(t, expected.Scheme, actual.Scheme)
	require.Equal(t, expected.Status, actual.Status)

	switch {
	case expected.SourceAccountID == nil && actual.SourceAccountID == nil:
	case expected.SourceAccountID != nil && actual.SourceAccountID != nil:
		require.Equal(t, *expected.SourceAccountID, *actual.SourceAccountID)
	default:
		require.Fail(t, "source account id mismatch")
	}

	switch {
	case expected.DestinationAccountID == nil && actual.DestinationAccountID == nil:
	case expected.DestinationAccountID != nil && actual.DestinationAccountID != nil:
		require.Equal(t, *expected.DestinationAccountID, *actual.DestinationAccountID)
	default:
		require.Fail(t, "destination account id mismatch")
	}

	require.Equal(t, len(expected.Metadata), len(actual.Metadata))
	for k, v := range expected.Metadata {
		_, ok := actual.Metadata[k]
		require.True(t, ok)
		require.Equal(t, v, actual.Metadata[k])
	}

	compareAdjustments(t, expected.Adjustments, actual.Adjustments)
}

func compareAdjustments(t *testing.T, expected, actual []models.PaymentAdjustment) {
	require.Equal(t, len(expected), len(actual))
	for i := range expected {
		require.Equal(t, expected[i].ID, actual[i].ID)
		require.Equal(t, expected[i].Reference, actual[i].Reference)
		require.Equal(t, expected[i].CreatedAt, actual[i].CreatedAt)
		require.Equal(t, expected[i].Status, actual[i].Status)
		require.Equal(t, expected[i].Amount, actual[i].Amount)
		require.Equal(t, expected[i].Asset, actual[i].Asset)

		require.Equal(t, len(expected[i].Metadata), len(actual[i].Metadata))
		for k, v := range expected[i].Metadata {
			_, ok := actual[i].Metadata[k]
			require.True(t, ok)
			require.Equal(t, v, actual[i].Metadata[k])
		}
	}
}

func TestFromPSPPayments(t *testing.T) {
	t.Parallel()

	// Given
	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	t.Run("valid payments", func(t *testing.T) {
		t.Parallel()
		
		// Given
		pspPayments := []models.PSPPayment{
			{
				Reference:     "payment1",
				CreatedAt:     now,
				Type:          models.PAYMENT_TYPE_PAYIN,
				Amount:        big.NewInt(100),
				Asset:         "USD/2",
				Scheme:        models.PAYMENT_SCHEME_OTHER,
				Status:        models.PAYMENT_STATUS_SUCCEEDED,
				Raw:           []byte(`{}`),
			},
			{
				Reference:     "payment2",
				CreatedAt:     now,
				Type:          models.PAYMENT_TYPE_PAYOUT,
				Amount:        big.NewInt(200),
				Asset:         "EUR/2",
				Scheme:        models.PAYMENT_SCHEME_OTHER,
				Status:        models.PAYMENT_STATUS_PENDING,
				Raw:           []byte(`{}`),
			},
		}
		
		payments, err := models.FromPSPPayments(pspPayments, connectorID)
		
		// Then
		require.NoError(t, err)
		assert.Len(t, payments, 2)
		assert.Equal(t, "payment1", payments[0].Reference)
		assert.Equal(t, "payment2", payments[1].Reference)
	})
	
	t.Run("invalid payment", func(t *testing.T) {
		t.Parallel()
		
		// Given
		invalidPayment := models.PSPPayment{
			CreatedAt: now,
			Type:      models.PAYMENT_TYPE_PAYIN,
			Amount:    big.NewInt(100),
			Asset:     "USD/2",
			Scheme:    models.PAYMENT_SCHEME_OTHER,
			Status:    models.PAYMENT_STATUS_SUCCEEDED,
			Raw:       []byte(`{}`),
		}
		pspPayments := []models.PSPPayment{invalidPayment}
		
		_, err := models.FromPSPPayments(pspPayments, connectorID)
		
		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reference is required")
	})
}

func TestPaymentMarshalUnmarshal(t *testing.T) {
	t.Parallel()

	// Given
	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	payment := models.Payment{
		ID: models.PaymentID{
			PaymentReference: models.PaymentReference{
				Reference: "payment123",
				Type:      models.PAYMENT_TYPE_PAYIN,
			},
			ConnectorID: connectorID,
		},
		ConnectorID:   connectorID,
		Reference:     "payment123",
		CreatedAt:     now,
		Type:          models.PAYMENT_TYPE_PAYIN,
		InitialAmount: big.NewInt(100),
		Amount:        big.NewInt(100),
		Asset:         "USD/2",
		Scheme:        models.PAYMENT_SCHEME_OTHER,
		Status:        models.PAYMENT_STATUS_SUCCEEDED,
		SourceAccountID: &models.AccountID{
			Reference:   "source123",
			ConnectorID: connectorID,
		},
		DestinationAccountID: &models.AccountID{
			Reference:   "dest123",
			ConnectorID: connectorID,
		},
		Metadata: map[string]string{
			"key": "value",
		},
		Adjustments: []models.PaymentAdjustment{
			{
				ID: models.PaymentAdjustmentID{
					PaymentID: models.PaymentID{
						PaymentReference: models.PaymentReference{
							Reference: "payment123",
							Type:      models.PAYMENT_TYPE_PAYIN,
						},
						ConnectorID: connectorID,
					},
					Reference: "adj123",
					CreatedAt: now,
					Status:    models.PAYMENT_STATUS_SUCCEEDED,
				},
				Reference: "adj123",
				CreatedAt: now,
				Status:    models.PAYMENT_STATUS_SUCCEEDED,
				Amount:    big.NewInt(100),
				Asset:     pointer.For("USD/2"),
				Metadata: map[string]string{
					"key": "value",
				},
				Raw: []byte(`{}`),
			},
		},
	}

	data, err := json.Marshal(payment)
	
	// Then
	require.NoError(t, err)

	var unmarshaledPayment models.Payment
	err = json.Unmarshal(data, &unmarshaledPayment)
	
	// Then
	require.NoError(t, err)
	comparePayment(t, payment, unmarshaledPayment)
}

func TestPSPPaymentValidate(t *testing.T) {
	t.Parallel()

	// Given
	now := time.Now().UTC()

	t.Run("valid payment", func(t *testing.T) {
		t.Parallel()
		
		// Given
		payment := models.PSPPayment{
			Reference: "payment123",
			CreatedAt: now,
			Type:      models.PAYMENT_TYPE_PAYIN,
			Amount:    big.NewInt(100),
			Asset:     "USD/2",
			Scheme:    models.PAYMENT_SCHEME_OTHER,
			Status:    models.PAYMENT_STATUS_SUCCEEDED,
			Raw:       []byte(`{}`),
		}
		
		err := payment.Validate()
		
		// Then
		assert.NoError(t, err)
	})

	t.Run("missing reference", func(t *testing.T) {
		t.Parallel()
		
		// Given
		payment := models.PSPPayment{
			CreatedAt: now,
			Type:      models.PAYMENT_TYPE_PAYIN,
			Amount:    big.NewInt(100),
			Asset:     "USD/2",
			Scheme:    models.PAYMENT_SCHEME_OTHER,
			Status:    models.PAYMENT_STATUS_SUCCEEDED,
			Raw:       []byte(`{}`),
		}
		
		err := payment.Validate()
		
		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reference is required")
	})

	t.Run("missing createdAt", func(t *testing.T) {
		t.Parallel()
		
		// Given
		payment := models.PSPPayment{
			Reference: "payment123",
			Type:      models.PAYMENT_TYPE_PAYIN,
			Amount:    big.NewInt(100),
			Asset:     "USD/2",
			Scheme:    models.PAYMENT_SCHEME_OTHER,
			Status:    models.PAYMENT_STATUS_SUCCEEDED,
			Raw:       []byte(`{}`),
		}
		
		err := payment.Validate()
		
		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "createdAt is required")
	})

	t.Run("missing type", func(t *testing.T) {
		t.Parallel()
		
		// Given
		payment := models.PSPPayment{
			Reference: "payment123",
			CreatedAt: now,
			Amount:    big.NewInt(100),
			Asset:     "USD/2",
			Scheme:    models.PAYMENT_SCHEME_OTHER,
			Status:    models.PAYMENT_STATUS_SUCCEEDED,
			Raw:       []byte(`{}`),
		}
		
		err := payment.Validate()
		
		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type is required")
	})

	t.Run("missing amount", func(t *testing.T) {
		t.Parallel()
		
		// Given
		payment := models.PSPPayment{
			Reference: "payment123",
			CreatedAt: now,
			Type:      models.PAYMENT_TYPE_PAYIN,
			Asset:     "USD/2",
			Scheme:    models.PAYMENT_SCHEME_OTHER,
			Status:    models.PAYMENT_STATUS_SUCCEEDED,
			Raw:       []byte(`{}`),
		}
		
		err := payment.Validate()
		
		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "amount is required")
	})

	t.Run("invalid asset", func(t *testing.T) {
		t.Parallel()
		
		// Given
		payment := models.PSPPayment{
			Reference: "payment123",
			CreatedAt: now,
			Type:      models.PAYMENT_TYPE_PAYIN,
			Amount:    big.NewInt(100),
			Asset:     "invalid",
			Scheme:    models.PAYMENT_SCHEME_OTHER,
			Status:    models.PAYMENT_STATUS_SUCCEEDED,
			Raw:       []byte(`{}`),
		}
		
		err := payment.Validate()
		
		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid asset")
	})

	t.Run("missing raw", func(t *testing.T) {
		t.Parallel()
		
		// Given
		payment := models.PSPPayment{
			Reference: "payment123",
			CreatedAt: now,
			Type:      models.PAYMENT_TYPE_PAYIN,
			Amount:    big.NewInt(100),
			Asset:     "USD/2",
			Scheme:    models.PAYMENT_SCHEME_OTHER,
			Status:    models.PAYMENT_STATUS_SUCCEEDED,
		}
		
		err := payment.Validate()
		
		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "raw is required")
	})
}

func TestPSPPaymentHasParent(t *testing.T) {
	t.Parallel()

	t.Run("with parent reference", func(t *testing.T) {
		t.Parallel()
		
		// Given
		payment := models.PSPPayment{
			ParentReference: "parent123",
			Reference:       "payment123",
		}
		
		hasParent := payment.HasParent()
		
		// Then
		assert.True(t, hasParent)
	})

	t.Run("without parent reference", func(t *testing.T) {
		t.Parallel()
		
		// Given
		payment := models.PSPPayment{
			Reference: "payment123",
		}
		
		hasParent := payment.HasParent()
		
		// Then
		assert.False(t, hasParent)
	})
}
