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
		
		// Given
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
			result := tc.paymentType.String()
			
			// Then
			assert.Equal(t, tc.expected, result)
		}
	})

	t.Run("PaymentTypeFromString", func(t *testing.T) {
		t.Parallel()
		
		// Given
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
			
			// Then
			if tc.hasError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unknown payment type")
				assert.Equal(t, models.PAYMENT_TYPE_UNKNOWN, paymentType)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, paymentType)
			}
		}
	})

	t.Run("MustPaymentTypeFromString", func(t *testing.T) {
		t.Parallel()
		
		// Given
		testCases := []struct {
			input    string
			expected models.PaymentType
		}{
			{"PAY-IN", models.PAYMENT_TYPE_PAYIN},
			{"PAYOUT", models.PAYMENT_TYPE_PAYOUT},
			{"TRANSFER", models.PAYMENT_TYPE_TRANSFER},
		}
		
		for _, tc := range testCases {
			result := models.MustPaymentTypeFromString(tc.input)
			
			// Then
			assert.Equal(t, tc.expected, result)
		}
	})

	t.Run("JSON", func(t *testing.T) {
		t.Parallel()
		
		// Given
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
			
			// Then
			require.NoError(t, err)
			assert.Equal(t, paymentType, unmarshaled)
		}
		
		// Given
		var paymentType models.PaymentType
		
		err := json.Unmarshal([]byte(`"INVALID"`), &paymentType)
		
		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown payment type")
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		
		// Given
		paymentType := models.PAYMENT_TYPE_PAYIN
		
		val, err := paymentType.Value()
		
		// Then
		require.NoError(t, err)
		assert.Equal(t, "PAY-IN", val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()
		
		t.Run("valid PAY-IN string", func(t *testing.T) {
			t.Parallel()
			
			// Given
			var paymentType1 models.PaymentType
			
			err := paymentType1.Scan("PAY-IN")
			
			// Then
			require.NoError(t, err)
			assert.Equal(t, models.PAYMENT_TYPE_PAYIN, paymentType1)
		})
		
		t.Run("valid PAYOUT string", func(t *testing.T) {
			t.Parallel()
			
			// Given
			var paymentType2 models.PaymentType
			
			err := paymentType2.Scan("PAYOUT")
			
			// Then
			require.NoError(t, err)
			assert.Equal(t, models.PAYMENT_TYPE_PAYOUT, paymentType2)
		})
		
		t.Run("invalid type", func(t *testing.T) {
			t.Parallel()
			
			// Given
			var paymentType3 models.PaymentType
			
			err := paymentType3.Scan(123)
			
			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unknown payment type")
		})
		
		t.Run("invalid string", func(t *testing.T) {
			t.Parallel()
			
			// Given
			var paymentType4 models.PaymentType
			
			err := paymentType4.Scan("INVALID")
			
			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unknown payment type")
		})
		
		t.Run("nil value", func(t *testing.T) {
			t.Parallel()
			
			// Given
			var paymentType5 models.PaymentType
			
			err := paymentType5.Scan(nil)
			
			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "payment type is nil")
		})
	})
}
