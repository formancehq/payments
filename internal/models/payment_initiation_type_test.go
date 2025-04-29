package models_test

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentInitiationType(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		// Given

		testCases := []struct {
			initType models.PaymentInitiationType
			expected string
		}{
			{models.PAYMENT_INITIATION_TYPE_UNKNOWN, "UNKNOWN"},
			{models.PAYMENT_INITIATION_TYPE_TRANSFER, "TRANSFER"},
			{models.PAYMENT_INITIATION_TYPE_PAYOUT, "PAYOUT"},
		}

		for _, tc := range testCases {
			result := tc.initType.String()

			// Then
			assert.Equal(t, tc.expected, result)
		}
	})

	t.Run("PaymentInitiationTypeFromString", func(t *testing.T) {
		t.Parallel()
		// Given

		testCases := []struct {
			input    string
			expected models.PaymentInitiationType
			hasError bool
		}{
			{"TRANSFER", models.PAYMENT_INITIATION_TYPE_TRANSFER, false},
			{"PAYOUT", models.PAYMENT_INITIATION_TYPE_PAYOUT, false},
			{"UNKNOWN", models.PAYMENT_INITIATION_TYPE_UNKNOWN, false},
			{"invalid", models.PAYMENT_INITIATION_TYPE_UNKNOWN, true},
			{"", models.PAYMENT_INITIATION_TYPE_UNKNOWN, true},
		}

		for _, tc := range testCases {
			initType, err := models.PaymentInitiationTypeFromString(tc.input)
			if tc.hasError {

				// Then
				assert.Error(t, err)
			} else {
				// Then
				require.NoError(t, err)
				assert.Equal(t, tc.expected, initType)
			}
		}
	})

	t.Run("MustPaymentInitiationTypeFromString", func(t *testing.T) {
		t.Parallel()
		// Given

		testCases := []struct {
			input    string
			expected models.PaymentInitiationType
		}{
			{"TRANSFER", models.PAYMENT_INITIATION_TYPE_TRANSFER},
			{"PAYOUT", models.PAYMENT_INITIATION_TYPE_PAYOUT},
			{"UNKNOWN", models.PAYMENT_INITIATION_TYPE_UNKNOWN},
		}

		for _, tc := range testCases {
			result := models.MustPaymentInitiationTypeFromString(tc.input)

			// Then
			assert.Equal(t, tc.expected, result)
		}

	})

	t.Run("JSON", func(t *testing.T) {
		t.Parallel()
		// Given

		types := []models.PaymentInitiationType{
			models.PAYMENT_INITIATION_TYPE_UNKNOWN,
			models.PAYMENT_INITIATION_TYPE_TRANSFER,
			models.PAYMENT_INITIATION_TYPE_PAYOUT,
		}

		for _, initType := range types {
			data, err := json.Marshal(initType)

			// Then
			require.NoError(t, err)

			var unmarshaled models.PaymentInitiationType
			err = json.Unmarshal(data, &unmarshaled)
			// Then
			require.NoError(t, err)

			assert.Equal(t, initType, unmarshaled)
		}

		var initType models.PaymentInitiationType
		err := json.Unmarshal([]byte(`"INVALID"`), &initType)
		// Then
		assert.Error(t, err)
	})

	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		// Given

		val, err := models.PAYMENT_INITIATION_TYPE_TRANSFER.Value()

		// Then
		require.NoError(t, err)
		assert.Equal(t, "TRANSFER", val)
	})

	t.Run("Scan", func(t *testing.T) {
		t.Parallel()
		// Given

		var initType models.PaymentInitiationType

		err := initType.Scan("TRANSFER")

		// Then
		require.NoError(t, err)
		assert.Equal(t, models.PAYMENT_INITIATION_TYPE_TRANSFER, initType)

		err = initType.Scan(nil)
		// Then
		assert.Error(t, err)

		err = initType.Scan(123)
		// Then
		assert.Error(t, err)

		err = initType.Scan("INVALID")
		// Then
		assert.Error(t, err)
	})
}
