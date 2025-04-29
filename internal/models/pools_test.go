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

func TestPoolMarshalUnmarshal(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	id := uuid.New()
	connectorID := models.ConnectorID{
		Provider:  "stripe",
		Reference: uuid.New(),
	}

	accountID1 := models.AccountID{
		Reference:   "acc1",
		ConnectorID: connectorID,
	}
	accountID2 := models.AccountID{
		Reference:   "acc2",
		ConnectorID: connectorID,
	}

	pool := models.Pool{
		ID:           id,
		Name:         "Test Pool",
		CreatedAt:    now,
		PoolAccounts: []models.AccountID{accountID1, accountID2},
	}

	data, err := json.Marshal(pool)
	// Then
	require.NoError(t, err)

	var unmarshaledPool models.Pool
	err = json.Unmarshal(data, &unmarshaledPool)
	// Then
	require.NoError(t, err)

	assert.Equal(t, pool.ID, unmarshaledPool.ID)
	assert.Equal(t, pool.Name, unmarshaledPool.Name)
	assert.Equal(t, pool.CreatedAt, unmarshaledPool.CreatedAt)
	assert.Len(t, unmarshaledPool.PoolAccounts, 2)
	assert.Equal(t, accountID1.Reference, unmarshaledPool.PoolAccounts[0].Reference)
	assert.Equal(t, accountID2.Reference, unmarshaledPool.PoolAccounts[1].Reference)

	invalidJSON := []byte(`{"id": "not-a-uuid", "name": "Invalid Pool", "createdAt": "2023-01-01T00:00:00Z", "poolAccounts": []}`)
	err = json.Unmarshal(invalidJSON, &unmarshaledPool)
	// Then
	assert.Error(t, err)

	invalidJSON = []byte(`{"id": "` + uuid.New().String() + `", "name": "Invalid Pool", "createdAt": "2023-01-01T00:00:00Z", "poolAccounts": ["invalid-account-id"]}`)
	err = json.Unmarshal(invalidJSON, &unmarshaledPool)
	// Then
	assert.Error(t, err)
}

func TestPoolIdempotencyKey(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	connectorID := models.ConnectorID{
		Provider:  "stripe",
		Reference: uuid.New(),
	}

	accountID1 := models.AccountID{
		Reference:   "acc1",
		ConnectorID: connectorID,
	}
	accountID2 := models.AccountID{
		Reference:   "acc2",
		ConnectorID: connectorID,
	}

	pool := models.Pool{
		ID:           id,
		Name:         "Test Pool",
		CreatedAt:    time.Now().UTC(),
		PoolAccounts: []models.AccountID{accountID1, accountID2},
	}

	key := pool.IdempotencyKey()
	assert.NotEmpty(t, key)

	key2 := pool.IdempotencyKey()
	assert.Equal(t, key, key2)

	pool2 := models.Pool{
		ID:           uuid.New(),
		Name:         "Test Pool 2",
		CreatedAt:    time.Now().UTC(),
		PoolAccounts: []models.AccountID{accountID1},
	}
	key3 := pool2.IdempotencyKey()
	assert.NotEqual(t, key, key3)
}
