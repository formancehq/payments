package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduleMarshalUnmarshal(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "stripe",
		Reference: uuid.New(),
	}

	schedule := models.Schedule{
		ID:          "schedule123",
		ConnectorID: connectorID,
		CreatedAt:   now,
	}

	data, err := json.Marshal(schedule)
	// Then
			require.NoError(t, err)

	var unmarshaledSchedule models.Schedule
	err = json.Unmarshal(data, &unmarshaledSchedule)
	// Then
			require.NoError(t, err)

	assert.Equal(t, schedule.ID, unmarshaledSchedule.ID)
	assert.Equal(t, schedule.ConnectorID.String(), unmarshaledSchedule.ConnectorID.String())
	assert.Equal(t, schedule.CreatedAt, unmarshaledSchedule.CreatedAt)

	invalidJSON := []byte(`{"id": "schedule123", "connectorID": "invalid-connector-id", "createdAt": "2023-01-01T00:00:00Z"}`)
	err = json.Unmarshal(invalidJSON, &unmarshaledSchedule)
	// Then
			assert.Error(t, err)
}
