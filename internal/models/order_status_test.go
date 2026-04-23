package models_test

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderStatus(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			status   models.OrderStatus
			expected string
		}{
			{models.ORDER_STATUS_UNKNOWN, "UNKNOWN"},
			{models.ORDER_STATUS_PENDING, "PENDING"},
			{models.ORDER_STATUS_OPEN, "OPEN"},
			{models.ORDER_STATUS_PARTIALLY_FILLED, "PARTIALLY_FILLED"},
			{models.ORDER_STATUS_FILLED, "FILLED"},
			{models.ORDER_STATUS_CANCELLED, "CANCELLED"},
			{models.ORDER_STATUS_FAILED, "FAILED"},
			{models.ORDER_STATUS_EXPIRED, "EXPIRED"},
			{models.OrderStatus(999), "UNKNOWN"},
		}
		for _, tc := range cases {
			assert.Equal(t, tc.expected, tc.status.String())
		}
	})

	t.Run("FromString", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			input    string
			expected models.OrderStatus
			hasError bool
		}{
			{"UNKNOWN", models.ORDER_STATUS_UNKNOWN, false},
			{"PENDING", models.ORDER_STATUS_PENDING, false},
			{"OPEN", models.ORDER_STATUS_OPEN, false},
			{"PARTIALLY_FILLED", models.ORDER_STATUS_PARTIALLY_FILLED, false},
			{"FILLED", models.ORDER_STATUS_FILLED, false},
			{"CANCELLED", models.ORDER_STATUS_CANCELLED, false},
			{"FAILED", models.ORDER_STATUS_FAILED, false},
			{"EXPIRED", models.ORDER_STATUS_EXPIRED, false},
			{"INVALID", models.ORDER_STATUS_UNKNOWN, true},
		}
		for _, tc := range cases {
			result, err := models.OrderStatusFromString(tc.input)
			if tc.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.expected, result)
		}
	})

	t.Run("MarshalJSON_UnmarshalJSON", func(t *testing.T) {
		t.Parallel()
		for _, s := range []models.OrderStatus{
			models.ORDER_STATUS_PENDING, models.ORDER_STATUS_OPEN,
			models.ORDER_STATUS_FILLED, models.ORDER_STATUS_CANCELLED,
		} {
			data, err := json.Marshal(s)
			require.NoError(t, err)

			var result models.OrderStatus
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)
			assert.Equal(t, s, result)
		}
	})

	t.Run("Value_Scan", func(t *testing.T) {
		t.Parallel()
		for _, s := range []models.OrderStatus{
			models.ORDER_STATUS_PENDING, models.ORDER_STATUS_FILLED,
		} {
			v, err := s.Value()
			require.NoError(t, err)

			var scanned models.OrderStatus
			err = scanned.Scan(v)
			require.NoError(t, err)
			assert.Equal(t, s, scanned)
		}

		_, err := models.ORDER_STATUS_UNKNOWN.Value()
		require.Error(t, err)
	})

	t.Run("IsFinal", func(t *testing.T) {
		t.Parallel()
		assert.False(t, models.ORDER_STATUS_PENDING.IsFinal())
		assert.False(t, models.ORDER_STATUS_OPEN.IsFinal())
		assert.False(t, models.ORDER_STATUS_PARTIALLY_FILLED.IsFinal())
		assert.True(t, models.ORDER_STATUS_FILLED.IsFinal())
		assert.True(t, models.ORDER_STATUS_CANCELLED.IsFinal())
		assert.True(t, models.ORDER_STATUS_FAILED.IsFinal())
		assert.True(t, models.ORDER_STATUS_EXPIRED.IsFinal())
	})

	t.Run("CanCancel", func(t *testing.T) {
		t.Parallel()
		assert.True(t, models.ORDER_STATUS_PENDING.CanCancel())
		assert.True(t, models.ORDER_STATUS_OPEN.CanCancel())
		assert.True(t, models.ORDER_STATUS_PARTIALLY_FILLED.CanCancel())
		assert.False(t, models.ORDER_STATUS_FILLED.CanCancel())
		assert.False(t, models.ORDER_STATUS_CANCELLED.CanCancel())
	})
}
