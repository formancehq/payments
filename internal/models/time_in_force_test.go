package models_test

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeInForce(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "UNKNOWN", models.TIME_IN_FORCE_UNKNOWN.String())
		assert.Equal(t, "GOOD_UNTIL_CANCELLED", models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED.String())
		assert.Equal(t, "GOOD_UNTIL_DATE_TIME", models.TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME.String())
		assert.Equal(t, "IMMEDIATE_OR_CANCEL", models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL.String())
		assert.Equal(t, "FILL_OR_KILL", models.TIME_IN_FORCE_FILL_OR_KILL.String())
		assert.Equal(t, "UNKNOWN", models.TimeInForce(999).String())
	})

	t.Run("FromString", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			input    string
			expected models.TimeInForce
			hasError bool
		}{
			{"UNKNOWN", models.TIME_IN_FORCE_UNKNOWN, false},
			{"GOOD_UNTIL_CANCELLED", models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED, false},
			{"GTC", models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED, false},
			{"GOOD_UNTIL_DATE_TIME", models.TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME, false},
			{"GTD", models.TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME, false},
			{"IMMEDIATE_OR_CANCEL", models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL, false},
			{"IOC", models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL, false},
			{"FILL_OR_KILL", models.TIME_IN_FORCE_FILL_OR_KILL, false},
			{"FOK", models.TIME_IN_FORCE_FILL_OR_KILL, false},
			{"INVALID", models.TIME_IN_FORCE_UNKNOWN, true},
		}
		for _, tc := range cases {
			result, err := models.TimeInForceFromString(tc.input)
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
		for _, tif := range []models.TimeInForce{
			models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
			models.TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME,
			models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL,
			models.TIME_IN_FORCE_FILL_OR_KILL,
		} {
			data, err := json.Marshal(tif)
			require.NoError(t, err)

			var result models.TimeInForce
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)
			assert.Equal(t, tif, result)
		}
	})

	t.Run("Value_Scan", func(t *testing.T) {
		t.Parallel()
		for _, tif := range []models.TimeInForce{
			models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
			models.TIME_IN_FORCE_FILL_OR_KILL,
		} {
			v, err := tif.Value()
			require.NoError(t, err)

			var scanned models.TimeInForce
			err = scanned.Scan(v)
			require.NoError(t, err)
			assert.Equal(t, tif, scanned)
		}
		_, err := models.TIME_IN_FORCE_UNKNOWN.Value()
		require.Error(t, err)
	})
}
