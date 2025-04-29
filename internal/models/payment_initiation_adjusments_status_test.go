package models_test

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentInitiationAdjustmentStatus(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		// Given

		testCases := []struct {
			status   models.PaymentInitiationAdjustmentStatus
			expected string
		}{
			{models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION, "WAITING_FOR_VALIDATION"},
			{models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING, "PROCESSING"},
			{models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED, "PROCESSED"},
			{models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED, "FAILED"},
			{models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REJECTED, "REJECTED"},
			{models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_PROCESSING, "REVERSE_PROCESSING"},
			{models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_FAILED, "REVERSE_FAILED"},
			{models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED, "REVERSED"},
			{models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_SCHEDULED_FOR_PROCESSING, "SCHEDULED_FOR_PROCESSING"},
			{models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_UNKNOWN, "UNKNOWN"},
		}

		for _, tc := range testCases {
			// When
			result := tc.status.String()

			// Then
			assert.Equal(t, tc.expected, result)
		}
	})

	t.Run("PaymentInitiationAdjustmentStatusFromString", func(t *testing.T) {
		t.Parallel()
		// Given

		testCases := []struct {
			input    string
			expected models.PaymentInitiationAdjustmentStatus
			hasError bool
		}{
			{"WAITING_FOR_VALIDATION", models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION, false},
			{"PROCESSING", models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING, false},
			{"PROCESSED", models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED, false},
			{"FAILED", models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED, false},
			{"REJECTED", models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REJECTED, false},
			{"REVERSE_PROCESSING", models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_PROCESSING, false},
			{"REVERSE_FAILED", models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_FAILED, false},
			{"REVERSED", models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED, false},
			{"SCHEDULED_FOR_PROCESSING", models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_SCHEDULED_FOR_PROCESSING, false},
			{"UNKNOWN", models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_UNKNOWN, false},
			{"invalid", models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_UNKNOWN, true},
			{"", models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_UNKNOWN, true},
		}

		for _, tc := range testCases {
			// When
			status, err := models.PaymentInitiationAdjustmentStatusFromString(tc.input)
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

		statuses := []models.PaymentInitiationAdjustmentStatus{
			models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION,
			models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING,
			models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
			models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
		}

		for _, status := range statuses {
			// When
			data, err := json.Marshal(status)

			// Then
			require.NoError(t, err)

			var unmarshaled models.PaymentInitiationAdjustmentStatus
			err = json.Unmarshal(data, &unmarshaled)
			// Then
			require.NoError(t, err)

			assert.Equal(t, status, unmarshaled)
		}

		var status models.PaymentInitiationAdjustmentStatus
		err := json.Unmarshal([]byte(`"INVALID"`), &status)
		// Then
		assert.Error(t, err)
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		// Given

		// When
		val, err := models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION.Value()

		// Then
		require.NoError(t, err)
		assert.Equal(t, "WAITING_FOR_VALIDATION", val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()
		// Given

		var status models.PaymentInitiationAdjustmentStatus

		// When
		err := status.Scan("WAITING_FOR_VALIDATION")

		// Then
		require.NoError(t, err)
		assert.Equal(t, models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION, status)

		err = status.Scan("PROCESSING")
		// Then
		require.NoError(t, err)
		assert.Equal(t, models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING, status)

		err = status.Scan(123)
		// Then
		assert.Error(t, err)

		err = status.Scan("INVALID")
		// Then
		assert.Error(t, err)
	})
}
