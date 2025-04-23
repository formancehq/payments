package models_test

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentType(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		
		testCases := []struct {
			paymentType models.PaymentType
			expected    string
		}{
			{models.PAYMENT_TYPE_PAYIN, "PAY-IN"},
			{models.PAYMENT_TYPE_PAYOUT, "PAYOUT"},
			{models.PAYMENT_TYPE_TRANSFER, "TRANSFER"},
			{models.PAYMENT_TYPE_UNKNOWN, "UNKNOWN"},
		}
		
		for _, tc := range testCases {
			assert.Equal(t, tc.expected, tc.paymentType.String())
		}
	})

	t.Run("PaymentTypeFromString", func(t *testing.T) {
		t.Parallel()
		
		testCases := []struct {
			input    string
			expected models.PaymentType
			hasError bool
		}{
			{"PAY-IN", models.PAYMENT_TYPE_PAYIN, false},
			{"PAYOUT", models.PAYMENT_TYPE_PAYOUT, false},
			{"TRANSFER", models.PAYMENT_TYPE_TRANSFER, false},
			{"UNKNOWN", models.PAYMENT_TYPE_UNKNOWN, false},
			{"invalid", models.PAYMENT_TYPE_UNKNOWN, true},
			{"", models.PAYMENT_TYPE_UNKNOWN, true},
		}
		
		for _, tc := range testCases {
			paymentType, err := models.PaymentTypeFromString(tc.input)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, paymentType)
			}
		}
	})

	t.Run("MustPaymentTypeFromString", func(t *testing.T) {
		t.Parallel()
		
		testCases := []struct {
			input    string
			expected models.PaymentType
		}{
			{"PAY-IN", models.PAYMENT_TYPE_PAYIN},
			{"PAYOUT", models.PAYMENT_TYPE_PAYOUT},
			{"TRANSFER", models.PAYMENT_TYPE_TRANSFER},
		}
		
		for _, tc := range testCases {
			assert.Equal(t, tc.expected, models.MustPaymentTypeFromString(tc.input))
		}
	})

	t.Run("JSON", func(t *testing.T) {
		t.Parallel()
		
		types := []models.PaymentType{
			models.PAYMENT_TYPE_PAYIN,
			models.PAYMENT_TYPE_PAYOUT,
			models.PAYMENT_TYPE_TRANSFER,
		}
		
		for _, paymentType := range types {
			data, err := json.Marshal(paymentType)
			require.NoError(t, err)
			
			var unmarshaled models.PaymentType
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)
			
			assert.Equal(t, paymentType, unmarshaled)
		}
		
		var paymentType models.PaymentType
		err := json.Unmarshal([]byte(`"INVALID"`), &paymentType)
		assert.Error(t, err)
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		
		val, err := models.PAYMENT_TYPE_PAYIN.Value()
		require.NoError(t, err)
		assert.Equal(t, "PAY-IN", val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()
		
		var paymentType models.PaymentType
		
		err := paymentType.Scan("PAY-IN")
		require.NoError(t, err)
		assert.Equal(t, models.PAYMENT_TYPE_PAYIN, paymentType)
		
		err = paymentType.Scan("PAYOUT")
		require.NoError(t, err)
		assert.Equal(t, models.PAYMENT_TYPE_PAYOUT, paymentType)
		
		err = paymentType.Scan(123)
		assert.Error(t, err)
		
		err = paymentType.Scan("INVALID")
		assert.Error(t, err)
	})
}
