package models_test

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapabilityString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		capability models.Capability
		expected   string
	}{
		{models.CAPABILITY_FETCH_ACCOUNTS, "FETCH_ACCOUNTS"},
		{models.CAPABILITY_FETCH_BALANCES, "FETCH_BALANCES"},
		{models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS, "FETCH_EXTERNAL_ACCOUNTS"},
		{models.CAPABILITY_FETCH_PAYMENTS, "FETCH_PAYMENTS"},
		{models.CAPABILITY_FETCH_OTHERS, "FETCH_OTHERS"},
		{models.CAPABILITY_CREATE_WEBHOOKS, "CREATE_WEBHOOKS"},
		{models.CAPABILITY_TRANSLATE_WEBHOOKS, "TRANSLATE_WEBHOOKS"},
		{models.CAPABILITY_CREATE_BANK_ACCOUNT, "CREATE_BANK_ACCOUNT"},
		{models.CAPABILITY_CREATE_TRANSFER, "CREATE_TRANSFER"},
		{models.CAPABILITY_CREATE_PAYOUT, "CREATE_PAYOUT"},
		{models.CAPABILITY_ALLOW_FORMANCE_ACCOUNT_CREATION, "ALLOW_FORMANCE_ACCOUNT_CREATION"},
		{models.CAPABILITY_ALLOW_FORMANCE_PAYMENT_CREATION, "ALLOW_FORMANCE_PAYMENT_CREATION"},
		{models.CAPABILITY_FETCH_UNKNOWN, "UNKNOWN"},
		{models.Capability(999), "UNKNOWN"}, // Unknown capability
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expected, tc.capability.String())
	}
}

func TestCapabilityValue(t *testing.T) {
	t.Parallel()

	t.Run("valid capabilities", func(t *testing.T) {
		t.Parallel()
		// Given

		testCases := []struct {
			capability models.Capability
			expected   string
		}{
			{models.CAPABILITY_FETCH_ACCOUNTS, "FETCH_ACCOUNTS"},
			{models.CAPABILITY_FETCH_BALANCES, "FETCH_BALANCES"},
			{models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS, "FETCH_EXTERNAL_ACCOUNTS"},
			{models.CAPABILITY_FETCH_PAYMENTS, "FETCH_PAYMENTS"},
			{models.CAPABILITY_FETCH_OTHERS, "FETCH_OTHERS"},
			{models.CAPABILITY_CREATE_WEBHOOKS, "CREATE_WEBHOOKS"},
			{models.CAPABILITY_TRANSLATE_WEBHOOKS, "TRANSLATE_WEBHOOKS"},
			{models.CAPABILITY_CREATE_BANK_ACCOUNT, "CREATE_BANK_ACCOUNT"},
			{models.CAPABILITY_CREATE_TRANSFER, "CREATE_TRANSFER"},
			{models.CAPABILITY_CREATE_PAYOUT, "CREATE_PAYOUT"},
			{models.CAPABILITY_ALLOW_FORMANCE_ACCOUNT_CREATION, "ALLOW_FORMANCE_ACCOUNT_CREATION"},
			{models.CAPABILITY_ALLOW_FORMANCE_PAYMENT_CREATION, "ALLOW_FORMANCE_PAYMENT_CREATION"},
		}

		for _, tc := range testCases {
			val, err := tc.capability.Value()
		// When/Then
			require.NoError(t, err)
			assert.Equal(t, tc.expected, val)
		}
	})

	t.Run("invalid capability", func(t *testing.T) {
		t.Parallel()
		// Given

		val, err := models.CAPABILITY_FETCH_UNKNOWN.Value()
		// When/Then
		require.Error(t, err)
		assert.Nil(t, val)
		assert.Contains(t, err.Error(), "unknown capability")

		val, err = models.Capability(999).Value()
		require.Error(t, err)
		assert.Nil(t, val)
		assert.Contains(t, err.Error(), "unknown capability")
	})
}

func TestCapabilityScan(t *testing.T) {
	t.Parallel()

	t.Run("valid capabilities", func(t *testing.T) {
		t.Parallel()
		// Given

		testCases := []struct {
			input    string
			expected models.Capability
		}{
			{"FETCH_ACCOUNTS", models.CAPABILITY_FETCH_ACCOUNTS},
			{"FETCH_BALANCES", models.CAPABILITY_FETCH_BALANCES},
			{"FETCH_EXTERNAL_ACCOUNTS", models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS},
			{"FETCH_PAYMENTS", models.CAPABILITY_FETCH_PAYMENTS},
			{"FETCH_OTHERS", models.CAPABILITY_FETCH_OTHERS},
			{"CREATE_WEBHOOKS", models.CAPABILITY_CREATE_WEBHOOKS},
			{"TRANSLATE_WEBHOOKS", models.CAPABILITY_TRANSLATE_WEBHOOKS},
			{"CREATE_BANK_ACCOUNT", models.CAPABILITY_CREATE_BANK_ACCOUNT},
			{"CREATE_TRANSFER", models.CAPABILITY_CREATE_TRANSFER},
			{"CREATE_PAYOUT", models.CAPABILITY_CREATE_PAYOUT},
			{"ALLOW_FORMANCE_ACCOUNT_CREATION", models.CAPABILITY_ALLOW_FORMANCE_ACCOUNT_CREATION},
			{"ALLOW_FORMANCE_PAYMENT_CREATION", models.CAPABILITY_ALLOW_FORMANCE_PAYMENT_CREATION},
		}

		for _, tc := range testCases {
			var capability models.Capability
			err := capability.Scan(tc.input)
		// When/Then
			require.NoError(t, err)
			assert.Equal(t, tc.expected, capability)
		}
	})

	t.Run("invalid inputs", func(t *testing.T) {
		t.Parallel()
		// Given

		var capability models.Capability

		err := capability.Scan(nil)
		// When/Then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "capability is nil")

		err = capability.Scan(123)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown capability")

		err = capability.Scan("UNKNOWN_CAPABILITY")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown capability")
	})
}
