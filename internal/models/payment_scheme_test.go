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
		// Given
		
		testCases := []struct {
			scheme   models.PaymentScheme
			expected string
		}{
			{models.PAYMENT_SCHEME_UNKNOWN, "UNKNOWN"},
			{models.PAYMENT_SCHEME_CARD_VISA, "CARD_VISA"},
			{models.PAYMENT_SCHEME_CARD_MASTERCARD, "CARD_MASTERCARD"},
			{models.PAYMENT_SCHEME_CARD_AMEX, "CARD_AMEX"},
			{models.PAYMENT_SCHEME_CARD_DINERS, "CARD_DINERS"},
			{models.PAYMENT_SCHEME_CARD_DISCOVER, "CARD_DISCOVER"},
			{models.PAYMENT_SCHEME_CARD_JCB, "CARD_JCB"},
			{models.PAYMENT_SCHEME_CARD_UNION_PAY, "CARD_UNION_PAY"},
			{models.PAYMENT_SCHEME_CARD_ALIPAY, "CARD_ALIPAY"},
			{models.PAYMENT_SCHEME_CARD_CUP, "CARD_CUP"},
			{models.PAYMENT_SCHEME_SEPA_DEBIT, "SEPA_DEBIT"},
			{models.PAYMENT_SCHEME_SEPA_CREDIT, "SEPA_CREDIT"},
			{models.PAYMENT_SCHEME_SEPA, "SEPA"},
			{models.PAYMENT_SCHEME_GOOGLE_PAY, "GOOGLE_PAY"},
			{models.PAYMENT_SCHEME_APPLE_PAY, "APPLE_PAY"},
			{models.PAYMENT_SCHEME_DOKU, "DOKU"},
			{models.PAYMENT_SCHEME_DRAGON_PAY, "DRAGON_PAY"},
			{models.PAYMENT_SCHEME_MAESTRO, "MAESTRO"},
			{models.PAYMENT_SCHEME_MOL_PAY, "MOL_PAY"},
			{models.PAYMENT_SCHEME_A2A, "A2A"},
			{models.PAYMENT_SCHEME_ACH_DEBIT, "ACH_DEBIT"},
			{models.PAYMENT_SCHEME_ACH, "ACH"},
			{models.PAYMENT_SCHEME_RTP, "RTP"},
			{models.PAYMENT_SCHEME_OTHER, "OTHER"},
			{models.PaymentScheme(999), "UNKNOWN"}, // Test default case
		}
		
		for _, tc := range testCases {
		// When/Then
			assert.Equal(t, tc.expected, tc.scheme.String())
		}
	})

	t.Run("PaymentSchemeFromString", func(t *testing.T) {
		t.Parallel()
		// Given
		
		testCases := []struct {
			input    string
			expected models.PaymentScheme
			hasError bool
		}{
			{"CARD_VISA", models.PAYMENT_SCHEME_CARD_VISA, false},
			{"CARD_MASTERCARD", models.PAYMENT_SCHEME_CARD_MASTERCARD, false},
			{"CARD_AMEX", models.PAYMENT_SCHEME_CARD_AMEX, false},
			{"CARD_DINERS", models.PAYMENT_SCHEME_CARD_DINERS, false},
			{"CARD_DISCOVER", models.PAYMENT_SCHEME_CARD_DISCOVER, false},
			{"CARD_JCB", models.PAYMENT_SCHEME_CARD_JCB, false},
			{"CARD_UNION_PAY", models.PAYMENT_SCHEME_CARD_UNION_PAY, false},
			{"CARD_ALIPAY", models.PAYMENT_SCHEME_CARD_ALIPAY, false},
			{"CARD_CUP", models.PAYMENT_SCHEME_CARD_CUP, false},
			{"SEPA_DEBIT", models.PAYMENT_SCHEME_SEPA_DEBIT, false},
			{"SEPA_CREDIT", models.PAYMENT_SCHEME_SEPA_CREDIT, false},
			{"SEPA", models.PAYMENT_SCHEME_SEPA, false},
			{"GOOGLE_PAY", models.PAYMENT_SCHEME_GOOGLE_PAY, false},
			{"APPLE_PAY", models.PAYMENT_SCHEME_APPLE_PAY, false},
			{"DOKU", models.PAYMENT_SCHEME_DOKU, false},
			{"DRAGON_PAY", models.PAYMENT_SCHEME_DRAGON_PAY, false},
			{"MAESTRO", models.PAYMENT_SCHEME_MAESTRO, false},
			{"MOL_PAY", models.PAYMENT_SCHEME_MOL_PAY, false},
			{"A2A", models.PAYMENT_SCHEME_A2A, false},
			{"ACH_DEBIT", models.PAYMENT_SCHEME_ACH_DEBIT, false},
			{"ACH", models.PAYMENT_SCHEME_ACH, false},
			{"RTP", models.PAYMENT_SCHEME_RTP, false},
			{"OTHER", models.PAYMENT_SCHEME_OTHER, false},
			{"UNKNOWN", models.PAYMENT_SCHEME_UNKNOWN, false},
			{"invalid", models.PAYMENT_SCHEME_UNKNOWN, true},
			{"", models.PAYMENT_SCHEME_UNKNOWN, true},
		}
		
		for _, tc := range testCases {
			scheme, err := models.PaymentSchemeFromString(tc.input)
			if tc.hasError {
		// When/Then
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, scheme)
			}
		}
	})

	t.Run("MustPaymentSchemeFromString", func(t *testing.T) {
		t.Parallel()
		// Given
		
		testCases := []struct {
			input    string
			expected models.PaymentScheme
		}{
			{"CARD_VISA", models.PAYMENT_SCHEME_CARD_VISA},
			{"CARD_MASTERCARD", models.PAYMENT_SCHEME_CARD_MASTERCARD},
			{"SEPA", models.PAYMENT_SCHEME_SEPA},
		}
		
		for _, tc := range testCases {
		// When/Then
			assert.Equal(t, tc.expected, models.MustPaymentSchemeFromString(tc.input))
		}
	})

	t.Run("JSON", func(t *testing.T) {
		t.Parallel()
		// Given
		
		schemes := []models.PaymentScheme{
			models.PAYMENT_SCHEME_CARD_VISA,
			models.PAYMENT_SCHEME_CARD_MASTERCARD,
			models.PAYMENT_SCHEME_SEPA,
		}
		
		for _, scheme := range schemes {
			data, err := json.Marshal(scheme)
		// When/Then
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
		// Given
		
		val, err := models.PAYMENT_SCHEME_CARD_VISA.Value()
		// When/Then
		require.NoError(t, err)
		assert.Equal(t, "CARD_VISA", val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()
		// Given
		
		var scheme models.PaymentScheme
		
		err := scheme.Scan("CARD_VISA")
		// When/Then
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
