package models_test

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentScheme(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		
		assert.Equal(t, "CARD_VISA", models.PAYMENT_SCHEME_CARD_VISA.String())
		assert.Equal(t, "CARD_MASTERCARD", models.PAYMENT_SCHEME_CARD_MASTERCARD.String())
		assert.Equal(t, "CARD_AMEX", models.PAYMENT_SCHEME_CARD_AMEX.String())
		assert.Equal(t, "SEPA", models.PAYMENT_SCHEME_SEPA.String())
		assert.Equal(t, "ACH", models.PAYMENT_SCHEME_ACH.String())
		assert.Equal(t, "UNKNOWN", models.PAYMENT_SCHEME_UNKNOWN.String())
		assert.Equal(t, "OTHER", models.PAYMENT_SCHEME_OTHER.String())
	})

	t.Run("PaymentSchemeFromString", func(t *testing.T) {
		t.Parallel()
		
		testCases := []struct {
			input    string
			expected models.PaymentScheme
			hasError bool
		}{
			{"CARD_VISA", models.PAYMENT_SCHEME_CARD_VISA, false},
			{"CARD_MASTERCARD", models.PAYMENT_SCHEME_CARD_MASTERCARD, false},
			{"CARD_AMEX", models.PAYMENT_SCHEME_CARD_AMEX, false},
			{"SEPA", models.PAYMENT_SCHEME_SEPA, false},
			{"ACH", models.PAYMENT_SCHEME_ACH, false},
			{"UNKNOWN", models.PAYMENT_SCHEME_UNKNOWN, false},
			{"OTHER", models.PAYMENT_SCHEME_OTHER, false},
			{"invalid", models.PAYMENT_SCHEME_UNKNOWN, true},
			{"", models.PAYMENT_SCHEME_UNKNOWN, true},
		}
		
		for _, tc := range testCases {
			scheme, err := models.PaymentSchemeFromString(tc.input)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, scheme)
			}
		}
	})

	t.Run("MustPaymentSchemeFromString", func(t *testing.T) {
		t.Parallel()
		
		testCases := []struct {
			input    string
			expected models.PaymentScheme
		}{
			{"CARD_VISA", models.PAYMENT_SCHEME_CARD_VISA},
			{"CARD_MASTERCARD", models.PAYMENT_SCHEME_CARD_MASTERCARD},
			{"SEPA", models.PAYMENT_SCHEME_SEPA},
		}
		
		for _, tc := range testCases {
			assert.Equal(t, tc.expected, models.MustPaymentSchemeFromString(tc.input))
		}
	})

	t.Run("JSON", func(t *testing.T) {
		t.Parallel()
		
		schemes := []models.PaymentScheme{
			models.PAYMENT_SCHEME_CARD_VISA,
			models.PAYMENT_SCHEME_CARD_MASTERCARD,
			models.PAYMENT_SCHEME_SEPA,
		}
		
		for _, scheme := range schemes {
			data, err := json.Marshal(scheme)
			require.NoError(t, err)
			
			var unmarshaled models.PaymentScheme
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)
			
			assert.Equal(t, scheme, unmarshaled)
		}
		
		var scheme models.PaymentScheme
		err := json.Unmarshal([]byte(`"INVALID"`), &scheme)
		assert.Error(t, err)
		
		err = json.Unmarshal([]byte(`{invalid}`), &scheme)
		assert.Error(t, err)
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		
		val, err := models.PAYMENT_SCHEME_CARD_VISA.Value()
		require.NoError(t, err)
		assert.Equal(t, "CARD_VISA", val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()
		
		var scheme models.PaymentScheme
		
		err := scheme.Scan("CARD_VISA")
		require.NoError(t, err)
		assert.Equal(t, models.PAYMENT_SCHEME_CARD_VISA, scheme)
		
		
		err = scheme.Scan(123)
		assert.Error(t, err)
		
		err = scheme.Scan("INVALID")
		assert.Error(t, err)
		
		err = scheme.Scan("")
		assert.Error(t, err)
	})
}
