package models_test

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPSPBalanceValidate(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	balance := models.PSPBalance{
		AccountReference: "acc123",
		CreatedAt:        now,
		Amount:           big.NewInt(100),
		Asset:            "USD/2",
	}
	assert.NoError(t, balance.Validate())

	balance = models.PSPBalance{
		CreatedAt: now,
		Amount:    big.NewInt(100),
		Asset:     "USD/2",
	}
	assert.Error(t, balance.Validate())

	balance = models.PSPBalance{
		AccountReference: "acc123",
		Amount:           big.NewInt(100),
		Asset:            "USD/2",
	}
	assert.Error(t, balance.Validate())

	balance = models.PSPBalance{
		AccountReference: "acc123",
		CreatedAt:        now,
		Asset:            "USD/2",
	}
	assert.Error(t, balance.Validate())

	balance = models.PSPBalance{
		AccountReference: "acc123",
		CreatedAt:        now,
		Amount:           big.NewInt(100),
		Asset:            "invalid",
	}
	assert.Error(t, balance.Validate())
}

func TestBalanceIdempotencyKey(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.New(),
	}

	balance := models.Balance{
		AccountID: models.AccountID{
			Reference:   "acc123",
			ConnectorID: connectorID,
		},
		CreatedAt:     now,
		LastUpdatedAt: now.Add(time.Hour),
		Asset:         "USD/2",
		Balance:       big.NewInt(100),
	}

	key := balance.IdempotencyKey()
	assert.NotEmpty(t, key)

	key2 := balance.IdempotencyKey()
	assert.Equal(t, key, key2)

	balance2 := models.Balance{
		AccountID: models.AccountID{
			Reference:   "acc456",
			ConnectorID: connectorID,
		},
		CreatedAt:     now,
		LastUpdatedAt: now.Add(time.Hour),
		Asset:         "USD/2",
		Balance:       big.NewInt(100),
	}
	key3 := balance2.IdempotencyKey()
	assert.NotEqual(t, key, key3)
}

func TestBalanceMarshalUnmarshal(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.New(),
	}
	psuId, _ := uuid.Parse("d5d4a5e1-1f02-4a5f-b9b5-518232fde991")
	openBankingConnectionID := "21"

	balance := models.Balance{
		AccountID: models.AccountID{
			Reference:   "acc123",
			ConnectorID: connectorID,
		},
		CreatedAt:               now,
		LastUpdatedAt:           now.Add(time.Hour),
		Asset:                   "USD/2",
		Balance:                 big.NewInt(100),
		PsuID:                   &psuId,
		OpenBankingConnectionID: &openBankingConnectionID,
	}

	data, err := json.Marshal(balance)

	// Then
	require.NoError(t, err)

	var unmarshaledBalance models.Balance
	err = json.Unmarshal(data, &unmarshaledBalance)

	// Then
	require.NoError(t, err)

	assert.Equal(t, balance.AccountID.String(), unmarshaledBalance.AccountID.String())
	assert.Equal(t, balance.CreatedAt.Format(time.RFC3339Nano), unmarshaledBalance.CreatedAt.Format(time.RFC3339Nano))
	assert.Equal(t, balance.LastUpdatedAt.Format(time.RFC3339Nano), unmarshaledBalance.LastUpdatedAt.Format(time.RFC3339Nano))
	assert.Equal(t, balance.Asset, unmarshaledBalance.Asset)
	assert.Equal(t, balance.Balance.String(), unmarshaledBalance.Balance.String())
	assert.Equal(t, balance.PsuID.String(), unmarshaledBalance.PsuID.String())
	assert.Equal(t, balance.OpenBankingConnectionID, unmarshaledBalance.OpenBankingConnectionID)

	invalidJSON := `{
		"accountID": "invalid-account-id",
		"createdAt": "` + now.Format(time.RFC3339Nano) + `",
		"lastUpdatedAt": "` + now.Add(time.Hour).Format(time.RFC3339Nano) + `",
		"asset": "USD/2",
		"balance": 100
	}`
	err = json.Unmarshal([]byte(invalidJSON), &unmarshaledBalance)

	// Then
	assert.Error(t, err)
}

func TestFromPSPBalance(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.New(),
	}

	pspBalance := models.PSPBalance{
		AccountReference: "acc123",
		CreatedAt:        now,
		Amount:           big.NewInt(100),
		Asset:            "USD/2",
	}

	balance, err := models.FromPSPBalance(pspBalance, connectorID, nil, nil)

	// Then
	require.NoError(t, err)

	assert.Equal(t, "acc123", balance.AccountID.Reference)
	assert.Equal(t, connectorID, balance.AccountID.ConnectorID)
	assert.Equal(t, now, balance.CreatedAt)
	assert.Equal(t, now, balance.LastUpdatedAt)
	assert.Equal(t, "USD/2", balance.Asset)
	assert.Equal(t, big.NewInt(100), balance.Balance)

	invalidPSPBalance := models.PSPBalance{
		CreatedAt: now,
		Amount:    big.NewInt(100),
		Asset:     "USD/2",
	}

	_, err = models.FromPSPBalance(invalidPSPBalance, connectorID, nil, nil)

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing account reference: validation error")
}

func TestFromPSPBalances(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.New(),
	}

	pspBalances := []models.PSPBalance{
		{
			AccountReference: "acc1",
			CreatedAt:        now,
			Amount:           big.NewInt(100),
			Asset:            "USD/2",
		},
		{
			AccountReference: "acc2",
			CreatedAt:        now,
			Amount:           big.NewInt(200),
			Asset:            "EUR/2",
		},
	}

	balances, err := models.FromPSPBalances(pspBalances, connectorID, nil, nil)

	// Then
	require.NoError(t, err)
	assert.Len(t, balances, 2)
	assert.Equal(t, "acc1", balances[0].AccountID.Reference)
	assert.Equal(t, "acc2", balances[1].AccountID.Reference)
	assert.Equal(t, "USD/2", balances[0].Asset)
	assert.Equal(t, "EUR/2", balances[1].Asset)

	invalidPSPBalances := append(pspBalances, models.PSPBalance{
		CreatedAt: now,
		Amount:    big.NewInt(100),
		Asset:     "USD/2",
	})
	_, err = models.FromPSPBalances(invalidPSPBalances, connectorID, nil, nil)

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing account reference: validation error")
}

func TestFromPSPBalancesWithPsuIDAndConnectionID(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.New(),
	}
	psuId, _ := uuid.Parse("d5d4a5e1-1f02-4a5f-b9b5-518232fde991")
	openBankingConnectionID := "21"

	pspBalance := models.PSPBalance{
		AccountReference: "acc1",
		CreatedAt:        now,
		Amount:           big.NewInt(100),
		Asset:            "USD/2",
	}

	balance, err := models.FromPSPBalance(pspBalance, connectorID, &psuId, &openBankingConnectionID)

	// Then
	require.NoError(t, err)
	assert.Equal(t, "acc1", balance.AccountID.Reference)
	assert.Equal(t, "USD/2", balance.Asset)
	assert.Equal(t, psuId.String(), balance.PsuID.String())
	assert.Equal(t, &openBankingConnectionID, balance.OpenBankingConnectionID)
}

func TestFromPSPBalance_PSUIDAndConnectionIDFromParameters(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.New(),
	}

	// Parameters should override PSPBalance.PsuID
	paramPsuId, _ := uuid.Parse("d5d4a5e1-1f02-4a5f-b9b5-518232fde991")
	pspPsuId, _ := uuid.Parse("a1b2c3d4-5e6f-7g8h-9i0j-k1l2m3n4o5p6")
	// OpenBankingConnectionID from parameters should override PSPBalance.OpenBankingConnectionID
	paramConnectionID := "param-connection-123"
	pspConnectionID := "psp-connection-456"

	pspBalance := models.PSPBalance{
		AccountReference:        "acc1",
		CreatedAt:               now,
		Amount:                  big.NewInt(100),
		Asset:                   "USD/2",
		PsuID:                   &pspPsuId, // This should be overridden
		OpenBankingConnectionID: &pspConnectionID,
	}

	balance, err := models.FromPSPBalance(pspBalance, connectorID, &paramPsuId, &paramConnectionID)

	// Then
	require.NoError(t, err)
	assert.Equal(t, paramPsuId.String(), balance.PsuID.String())
	assert.Equal(t, paramConnectionID, *balance.OpenBankingConnectionID)
}

func TestFromPSPBalance_PSUIDFromPSPBalance(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.New(),
	}

	// PSUID should be taken from PSPBalance when not passed as parameter
	pspPsuId, _ := uuid.Parse("a1b2c3d4-5e6f-7g8h-9i0j-k1l2m3n4o5p6")
	pspConnectionID := "psp-connection-456"

	pspBalance := models.PSPBalance{
		AccountReference:        "acc1",
		CreatedAt:               now,
		Amount:                  big.NewInt(100),
		Asset:                   "USD/2",
		PsuID:                   &pspPsuId,
		OpenBankingConnectionID: &pspConnectionID,
	}

	balance, err := models.FromPSPBalance(pspBalance, connectorID, nil, nil)

	// Then
	require.NoError(t, err)
	assert.Equal(t, pspPsuId.String(), balance.PsuID.String())
	assert.Equal(t, &pspConnectionID, balance.OpenBankingConnectionID)
}

func TestFromPSPBalance_NoPSUIDOrConnectionID(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.New(),
	}

	// When neither parameters nor PSPBalance have PSUID/OpenBankingConnectionID, they should be nil
	pspBalance := models.PSPBalance{
		AccountReference: "acc1",
		CreatedAt:        now,
		Amount:           big.NewInt(100),
		Asset:            "USD/2",
		// No PsuID or OpenBankingConnectionID
	}

	balance, err := models.FromPSPBalance(pspBalance, connectorID, nil, nil)

	// Then
	require.NoError(t, err)
	assert.Nil(t, balance.PsuID)
	assert.Nil(t, balance.OpenBankingConnectionID)
}
