package models_test

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderType(t *testing.T) {
	t.Parallel()

	allTypes := []struct {
		typ models.OrderType
		str string
	}{
		{models.ORDER_TYPE_UNKNOWN, "UNKNOWN"},
		{models.ORDER_TYPE_MARKET, "MARKET"},
		{models.ORDER_TYPE_LIMIT, "LIMIT"},
		{models.ORDER_TYPE_STOP_LIMIT, "STOP_LIMIT"},
		{models.ORDER_TYPE_STOP, "STOP"},
		{models.ORDER_TYPE_TWAP, "TWAP"},
		{models.ORDER_TYPE_VWAP, "VWAP"},
		{models.ORDER_TYPE_PEG, "PEG"},
		{models.ORDER_TYPE_BLOCK, "BLOCK"},
		{models.ORDER_TYPE_RFQ, "RFQ"},
		{models.ORDER_TYPE_TRAILING_STOP, "TRAILING_STOP"},
		{models.ORDER_TYPE_TRAILING_STOP_LIMIT, "TRAILING_STOP_LIMIT"},
		{models.ORDER_TYPE_TAKE_PROFIT, "TAKE_PROFIT"},
		{models.ORDER_TYPE_TAKE_PROFIT_LIMIT, "TAKE_PROFIT_LIMIT"},
		{models.ORDER_TYPE_LIMIT_MAKER, "LIMIT_MAKER"},
	}

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		for _, tc := range allTypes {
			assert.Equal(t, tc.str, tc.typ.String())
		}
		assert.Equal(t, "UNKNOWN", models.OrderType(999).String())
	})

	t.Run("FromString", func(t *testing.T) {
		t.Parallel()
		for _, tc := range allTypes {
			result, err := models.OrderTypeFromString(tc.str)
			require.NoError(t, err)
			assert.Equal(t, tc.typ, result)
		}
		_, err := models.OrderTypeFromString("INVALID_TYPE")
		require.Error(t, err)
	})

	t.Run("MarshalJSON_UnmarshalJSON", func(t *testing.T) {
		t.Parallel()
		for _, tc := range allTypes[1:] { // skip UNKNOWN
			data, err := json.Marshal(tc.typ)
			require.NoError(t, err)

			var result models.OrderType
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)
			assert.Equal(t, tc.typ, result)
		}
	})

	t.Run("Value_Scan", func(t *testing.T) {
		t.Parallel()
		for _, tc := range allTypes[1:] { // skip UNKNOWN
			v, err := tc.typ.Value()
			require.NoError(t, err)

			var scanned models.OrderType
			err = scanned.Scan(v)
			require.NoError(t, err)
			assert.Equal(t, tc.typ, scanned)
		}
		_, err := models.ORDER_TYPE_UNKNOWN.Value()
		require.Error(t, err)
	})
}
