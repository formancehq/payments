package models_test

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentStatus(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		
		testCases := []struct {
			status   models.PaymentStatus
			expected string
		}{
			{models.PAYMENT_STATUS_PENDING, "PENDING"},
			{models.PAYMENT_STATUS_SUCCEEDED, "SUCCEEDED"},
			{models.PAYMENT_STATUS_FAILED, "FAILED"},
			{models.PAYMENT_STATUS_CANCELLED, "CANCELLED"},
			{models.PAYMENT_STATUS_EXPIRED, "EXPIRED"},
			{models.PAYMENT_STATUS_AUTHORISATION, "AUTHORISATION"},
			{models.PAYMENT_STATUS_UNKNOWN, "UNKNOWN"},
		}
		
		for _, tc := range testCases {
			assert.Equal(t, tc.expected, tc.status.String())
		}
	})

	t.Run("PaymentStatusFromString", func(t *testing.T) {
		t.Parallel()
		
		testCases := []struct {
			input    string
			expected models.PaymentStatus
			hasError bool
		}{
			{"PENDING", models.PAYMENT_STATUS_PENDING, false},
			{"SUCCEEDED", models.PAYMENT_STATUS_SUCCEEDED, false},
			{"FAILED", models.PAYMENT_STATUS_FAILED, false},
			{"CANCELLED", models.PAYMENT_STATUS_CANCELLED, false},
			{"EXPIRED", models.PAYMENT_STATUS_EXPIRED, false},
			{"AUTHORISATION", models.PAYMENT_STATUS_AUTHORISATION, false},
			{"UNKNOWN", models.PAYMENT_STATUS_UNKNOWN, false},
			{"invalid", models.PAYMENT_STATUS_UNKNOWN, true},
			{"", models.PAYMENT_STATUS_UNKNOWN, true},
		}
		
		for _, tc := range testCases {
			status, err := models.PaymentStatusFromString(tc.input)
			if tc.hasError {
				assert.Error(t, err)
				assert.Equal(t, models.PAYMENT_STATUS_UNKNOWN, status)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, status)
			}
		}
	})

	t.Run("MustPaymentStatusFromString", func(t *testing.T) {
		t.Parallel()
		
		testCases := []struct {
			input    string
			expected models.PaymentStatus
		}{
			{"PENDING", models.PAYMENT_STATUS_PENDING},
			{"SUCCEEDED", models.PAYMENT_STATUS_SUCCEEDED},
			{"FAILED", models.PAYMENT_STATUS_FAILED},
		}
		
		for _, tc := range testCases {
			assert.Equal(t, tc.expected, models.MustPaymentStatusFromString(tc.input))
		}
	})

	t.Run("JSON", func(t *testing.T) {
		t.Parallel()
		
		statuses := []models.PaymentStatus{
			models.PAYMENT_STATUS_PENDING,
			models.PAYMENT_STATUS_SUCCEEDED,
			models.PAYMENT_STATUS_FAILED,
		}
		
		for _, status := range statuses {
			data, err := json.Marshal(status)
			require.NoError(t, err)
			
			var unmarshaled models.PaymentStatus
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)
			
			assert.Equal(t, status, unmarshaled)
		}
		
		var status models.PaymentStatus
		err := json.Unmarshal([]byte(`"INVALID"`), &status)
		assert.Error(t, err) // Should error for invalid status
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		
		val, err := models.PAYMENT_STATUS_SUCCEEDED.Value()
		require.NoError(t, err)
		assert.Equal(t, "SUCCEEDED", val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()
		
		var status models.PaymentStatus
		
		err := status.Scan("SUCCEEDED")
		require.NoError(t, err)
		assert.Equal(t, models.PAYMENT_STATUS_SUCCEEDED, status)
		
		
		err = status.Scan(123)
		assert.Error(t, err)
	})
}
