package models_test

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPSPOther(t *testing.T) {
	t.Parallel()

	t.Run("create and marshal PSPOther", func(t *testing.T) {
		t.Parallel()
		other := models.PSPOther{
			ID:    "other123",
			Other: json.RawMessage(`{"key": "value"}`),
		}

		data, err := json.Marshal(other)

		require.NoError(t, err)
		var unmarshaled models.PSPOther
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, other.ID, unmarshaled.ID)
		assert.JSONEq(t, string(other.Other), string(unmarshaled.Other))
	})

	t.Run("unmarshal PSPOther", func(t *testing.T) {
		t.Parallel()
		jsonData := `{"id": "other123", "other": {"key": "value"}}`

		var other models.PSPOther
		err := json.Unmarshal([]byte(jsonData), &other)

		require.NoError(t, err)
		assert.Equal(t, "other123", other.ID)
		assert.JSONEq(t, `{"key": "value"}`, string(other.Other))
	})

	t.Run("unmarshal PSPOther with null other", func(t *testing.T) {
		t.Parallel()
		jsonData := `{"id": "other123", "other": null}`

		var other models.PSPOther
		err := json.Unmarshal([]byte(jsonData), &other)

		require.NoError(t, err)
		assert.Equal(t, "other123", other.ID)
		assert.Equal(t, "null", string(other.Other))
	})
}
