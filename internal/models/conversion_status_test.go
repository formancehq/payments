package models_test

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConversionStatus(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "UNKNOWN", models.CONVERSION_STATUS_UNKNOWN.String())
		assert.Equal(t, "PENDING", models.CONVERSION_STATUS_PENDING.String())
		assert.Equal(t, "COMPLETED", models.CONVERSION_STATUS_COMPLETED.String())
		assert.Equal(t, "FAILED", models.CONVERSION_STATUS_FAILED.String())
		assert.Equal(t, "UNKNOWN", models.ConversionStatus(999).String())
	})

	t.Run("FromString", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			input    string
			expected models.ConversionStatus
			hasError bool
		}{
			{"UNKNOWN", models.CONVERSION_STATUS_UNKNOWN, false},
			{"PENDING", models.CONVERSION_STATUS_PENDING, false},
			{"COMPLETED", models.CONVERSION_STATUS_COMPLETED, false},
			{"FAILED", models.CONVERSION_STATUS_FAILED, false},
			{"INVALID", models.CONVERSION_STATUS_UNKNOWN, true},
		}
		for _, tc := range cases {
			result, err := models.ConversionStatusFromString(tc.input)
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
		for _, s := range []models.ConversionStatus{
			models.CONVERSION_STATUS_PENDING,
			models.CONVERSION_STATUS_COMPLETED,
			models.CONVERSION_STATUS_FAILED,
		} {
			data, err := json.Marshal(s)
			require.NoError(t, err)

			var result models.ConversionStatus
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)
			assert.Equal(t, s, result)
		}
	})

	t.Run("Value_Scan", func(t *testing.T) {
		t.Parallel()
		for _, s := range []models.ConversionStatus{
			models.CONVERSION_STATUS_PENDING,
			models.CONVERSION_STATUS_COMPLETED,
		} {
			v, err := s.Value()
			require.NoError(t, err)

			var scanned models.ConversionStatus
			err = scanned.Scan(v)
			require.NoError(t, err)
			assert.Equal(t, s, scanned)
		}
		_, err := models.CONVERSION_STATUS_UNKNOWN.Value()
		require.Error(t, err)
	})

	t.Run("IsFinal", func(t *testing.T) {
		t.Parallel()
		assert.False(t, models.CONVERSION_STATUS_PENDING.IsFinal())
		assert.True(t, models.CONVERSION_STATUS_COMPLETED.IsFinal())
		assert.True(t, models.CONVERSION_STATUS_FAILED.IsFinal())
	})
}
