package models_test

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderDirection(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "UNKNOWN", models.ORDER_DIRECTION_UNKNOWN.String())
		assert.Equal(t, "BUY", models.ORDER_DIRECTION_BUY.String())
		assert.Equal(t, "SELL", models.ORDER_DIRECTION_SELL.String())
		assert.Equal(t, "UNKNOWN", models.OrderDirection(999).String())
	})

	t.Run("FromString", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			input    string
			expected models.OrderDirection
			hasError bool
		}{
			{"UNKNOWN", models.ORDER_DIRECTION_UNKNOWN, false},
			{"BUY", models.ORDER_DIRECTION_BUY, false},
			{"SELL", models.ORDER_DIRECTION_SELL, false},
			{"INVALID", models.ORDER_DIRECTION_UNKNOWN, true},
		}
		for _, tc := range cases {
			result, err := models.OrderDirectionFromString(tc.input)
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
		for _, d := range []models.OrderDirection{models.ORDER_DIRECTION_BUY, models.ORDER_DIRECTION_SELL} {
			data, err := json.Marshal(d)
			require.NoError(t, err)

			var result models.OrderDirection
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)
			assert.Equal(t, d, result)
		}
	})

	t.Run("Value_Scan", func(t *testing.T) {
		t.Parallel()
		for _, d := range []models.OrderDirection{models.ORDER_DIRECTION_BUY, models.ORDER_DIRECTION_SELL} {
			v, err := d.Value()
			require.NoError(t, err)

			var scanned models.OrderDirection
			err = scanned.Scan(v)
			require.NoError(t, err)
			assert.Equal(t, d, scanned)
		}
		_, err := models.ORDER_DIRECTION_UNKNOWN.Value()
		require.Error(t, err)
	})
}
