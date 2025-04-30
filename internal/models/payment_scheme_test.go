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
			// When
			result := tc.scheme.String()

			// Then
			assert.Equal(t, tc.expected, result)
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
			// When
			scheme, err := models.PaymentSchemeFromString(tc.input)

			// Then
			if tc.hasError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unknown payment scheme")
				assert.Equal(t, models.PAYMENT_SCHEME_UNKNOWN, scheme)
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
			// When
			result := models.MustPaymentSchemeFromString(tc.input)

			// Then
			assert.Equal(t, tc.expected, result)
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
			// When
			data, err := json.Marshal(scheme)
			require.NoError(t, err)

			var unmarshaled models.PaymentScheme
			err = json.Unmarshal(data, &unmarshaled)

			// Then
			require.NoError(t, err)
			assert.Equal(t, scheme, unmarshaled)
		}

		t.Run("invalid string", func(t *testing.T) {
			t.Parallel()

			// Given
			var scheme1 models.PaymentScheme

			// When
			err := json.Unmarshal([]byte(`"INVALID"`), &scheme1)

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unknown payment scheme")
		})

		t.Run("invalid JSON", func(t *testing.T) {
			t.Parallel()

			// Given
			var scheme2 models.PaymentScheme

			// When
			err := json.Unmarshal([]byte(`{invalid}`), &scheme2)

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid character")
		})
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()

		// Given
		scheme := models.PAYMENT_SCHEME_CARD_VISA

		// When
		val, err := scheme.Value()

		// Then
		require.NoError(t, err)
		assert.Equal(t, "CARD_VISA", val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()

		t.Run("valid string", func(t *testing.T) {
			t.Parallel()

			// Given
			var scheme1 models.PaymentScheme

			// When
			err := scheme1.Scan("CARD_VISA")

			// Then
			require.NoError(t, err)
			assert.Equal(t, models.PAYMENT_SCHEME_CARD_VISA, scheme1)
		})

		t.Run("invalid type", func(t *testing.T) {
			t.Parallel()

			// Given
			var scheme2 models.PaymentScheme

			// When
			err := scheme2.Scan(123)

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unknown payment scheme")
		})

		t.Run("invalid string", func(t *testing.T) {
			t.Parallel()

			// Given
			var scheme3 models.PaymentScheme

			// When
			err := scheme3.Scan("INVALID")

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unknown payment scheme")
		})

		t.Run("empty string", func(t *testing.T) {
			t.Parallel()

			// Given
			var scheme4 models.PaymentScheme

			// When
			err := scheme4.Scan("")

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unknown payment scheme")
		})

		t.Run("nil value", func(t *testing.T) {
			t.Parallel()

			// Given
			var scheme5 models.PaymentScheme

			// When
			err := scheme5.Scan(nil)

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "payment type is nil")
		})
	})
}
