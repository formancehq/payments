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

func TestInstanceMarshalUnmarshal(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	terminatedAt := now.Add(time.Hour)
	errorMsg := "test error"
	connectorID := models.ConnectorID{
		Provider:  "stripe",
		Reference: uuid.New(),
	}

	instance := models.Instance{
		ID:           "instance123",
		ScheduleID:   "schedule123",
		ConnectorID:  connectorID,
		CreatedAt:    now,
		UpdatedAt:    now.Add(time.Minute),
		Terminated:   true,
		TerminatedAt: &terminatedAt,
		Error:        &errorMsg,
	}

	data, err := json.Marshal(instance)
	// Then
			require.NoError(t, err)

	var unmarshaledInstance models.Instance
	err = json.Unmarshal(data, &unmarshaledInstance)
	// Then
			require.NoError(t, err)

	assert.Equal(t, instance.ID, unmarshaledInstance.ID)
	assert.Equal(t, instance.ScheduleID, unmarshaledInstance.ScheduleID)
	assert.Equal(t, instance.ConnectorID.String(), unmarshaledInstance.ConnectorID.String())
	assert.Equal(t, instance.CreatedAt, unmarshaledInstance.CreatedAt)
	assert.Equal(t, instance.UpdatedAt, unmarshaledInstance.UpdatedAt)
	assert.Equal(t, instance.Terminated, unmarshaledInstance.Terminated)
	assert.Equal(t, instance.TerminatedAt.Format(time.RFC3339), unmarshaledInstance.TerminatedAt.Format(time.RFC3339))
	assert.Equal(t, *instance.Error, *unmarshaledInstance.Error)

	instance = models.Instance{
		ID:          "instance123",
		ScheduleID:  "schedule123",
		ConnectorID: connectorID,
		CreatedAt:   now,
		UpdatedAt:   now,
		Terminated:  false,
	}

	data, err = json.Marshal(instance)
	// Then
			require.NoError(t, err)

	err = json.Unmarshal(data, &unmarshaledInstance)
	// Then
			require.NoError(t, err)

	assert.Equal(t, instance.ID, unmarshaledInstance.ID)
	assert.Equal(t, instance.ScheduleID, unmarshaledInstance.ScheduleID)
	assert.Equal(t, instance.ConnectorID.String(), unmarshaledInstance.ConnectorID.String())
	assert.Equal(t, instance.CreatedAt, unmarshaledInstance.CreatedAt)
	assert.Equal(t, instance.UpdatedAt, unmarshaledInstance.UpdatedAt)
	assert.Equal(t, instance.Terminated, unmarshaledInstance.Terminated)
	assert.Nil(t, unmarshaledInstance.TerminatedAt)
	assert.Nil(t, unmarshaledInstance.Error)

	invalidJSON := []byte(`{"id": "instance123", "scheduleID": "schedule123", "connectorID": "invalid-connector-id", "createdAt": "2023-01-01T00:00:00Z", "updatedAt": "2023-01-01T00:00:00Z", "terminated": false}`)
	err = json.Unmarshal(invalidJSON, &unmarshaledInstance)
	// Then
			assert.Error(t, err)
}
