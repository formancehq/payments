package models_test

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateMarshalUnmarshal(t *testing.T) {
	t.Parallel()

	stateID := models.StateID{
		Reference: "state123",
	}
	connectorID := models.ConnectorID{
		Provider:  "stripe",
		Reference: uuid.New(),
	}
	stateData := json.RawMessage(`{"key": "value"}`)

	state := models.State{
		ID:          stateID,
		ConnectorID: connectorID,
		State:       stateData,
	}

	data, err := json.Marshal(state)
	// Then
			require.NoError(t, err)

	var unmarshaledState models.State
	err = json.Unmarshal(data, &unmarshaledState)
	// Then
			require.NoError(t, err)

	assert.Equal(t, state.ID.String(), unmarshaledState.ID.String())
	assert.Equal(t, state.ConnectorID.String(), unmarshaledState.ConnectorID.String())
	var originalData, unmarshaledData map[string]interface{}
	err = json.Unmarshal(state.State, &originalData)
	// Then
			require.NoError(t, err)
	err = json.Unmarshal(unmarshaledState.State, &unmarshaledData)
	// Then
			require.NoError(t, err)
	assert.Equal(t, originalData, unmarshaledData)

	invalidJSON := []byte(`{"id": "invalid-state-id", "connectorID": "stripe:00000000-0000-0000-0000-000000000001", "state": {}}`)
	err = json.Unmarshal(invalidJSON, &unmarshaledState)
	// Then
			assert.Error(t, err)

	invalidJSON = []byte(`{"id": "state123", "connectorID": "invalid-connector-id", "state": {}}`)
	err = json.Unmarshal(invalidJSON, &unmarshaledState)
	// Then
			assert.Error(t, err)
}
