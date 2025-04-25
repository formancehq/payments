package models_test

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentInitiationReversalAdjustmentStatus(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		// Given
		
		testCases := []struct {
			status   models.PaymentInitiationReversalAdjustmentStatus
			expected string
		}{
			{models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSING, "PROCESSING"},
			{models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED, "PROCESSED"},
			{models.PAYMENT_INITIATION_REVERSAL_STATUS_FAILED, "FAILED"},
			{models.PAYMENT_INITIATION_REVERSAL_STATUS_UNKNOWN, "UNKNOWN"},
		}
		
		for _, tc := range testCases {
			// When
			result := tc.status.String()
			
			// Then
			assert.Equal(t, tc.expected, result)
		}
	})

	t.Run("PaymentInitiationReversalStatusFromString", func(t *testing.T) {
		t.Parallel()
		// Given
		
		testCases := []struct {
			input    string
			expected models.PaymentInitiationReversalAdjustmentStatus
			hasError bool
		}{
			{"PROCESSING", models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSING, false},
			{"PROCESSED", models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED, false},
			{"FAILED", models.PAYMENT_INITIATION_REVERSAL_STATUS_FAILED, false},
			{"UNKNOWN", models.PAYMENT_INITIATION_REVERSAL_STATUS_UNKNOWN, false},
			{"invalid", models.PAYMENT_INITIATION_REVERSAL_STATUS_UNKNOWN, false}, // Note: This doesn't return an error in the implementation
			{"", models.PAYMENT_INITIATION_REVERSAL_STATUS_UNKNOWN, false},        // Note: This doesn't return an error in the implementation
		}
		
		for _, tc := range testCases {
			// When
			status, err := models.PaymentInitiationReversalStatusFromString(tc.input)
			if tc.hasError {
				// Then
				assert.Error(t, err)
			} else {
				// Then
			require.NoError(t, err)
				assert.Equal(t, tc.expected, status)
			}
		}
	})


	t.Run("JSON", func(t *testing.T) {
		t.Parallel()
		// Given
		
		statuses := []models.PaymentInitiationReversalAdjustmentStatus{
			models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSING,
			models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED,
			models.PAYMENT_INITIATION_REVERSAL_STATUS_FAILED,
		}
		
		for _, status := range statuses {
			// When
			data, err := json.Marshal(status)
			
			// Then
			require.NoError(t, err)
			
			var unmarshaled models.PaymentInitiationReversalAdjustmentStatus
			err = json.Unmarshal(data, &unmarshaled)
			// Then
			require.NoError(t, err)
			
			assert.Equal(t, status, unmarshaled)
		}
		
		var status models.PaymentInitiationReversalAdjustmentStatus
		err := json.Unmarshal([]byte(`"INVALID"`), &status)
		// Then
			require.NoError(t, err) // Note: This doesn't return an error in the implementation
		assert.Equal(t, models.PAYMENT_INITIATION_REVERSAL_STATUS_UNKNOWN, status)
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		// Given
		
		// When
		val, err := models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSING.Value()
		
		// Then
		require.NoError(t, err)
		assert.Equal(t, "PROCESSING", val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()
		// Given
		
		var status models.PaymentInitiationReversalAdjustmentStatus
		
		// When
		err := status.Scan("PROCESSING")
		
		// Then
		require.NoError(t, err)
		assert.Equal(t, models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSING, status)
		
		err = status.Scan("PROCESSED")
		// Then
			require.NoError(t, err)
		assert.Equal(t, models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED, status)
		
		err = status.Scan(123)
		assert.NoError(t, err)
		
		err = status.Scan("INVALID")
		assert.NoError(t, err) // Changed from require.NoError to assert.NoError
		assert.Equal(t, models.PAYMENT_INITIATION_REVERSAL_STATUS_UNKNOWN, status)
	})
}
