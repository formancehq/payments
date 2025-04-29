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

func TestBankAccountRelatedAccountMarshalJSON(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}

	accountID := models.AccountID{
		Reference:   "acc123",
		ConnectorID: connectorID,
	}

	relatedAccount := models.BankAccountRelatedAccount{
		AccountID: accountID,
		CreatedAt: now,
	}

	data, err := json.Marshal(relatedAccount)
	// Then
	require.NoError(t, err)

	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	// Then
	require.NoError(t, err)

	assert.Equal(t, accountID.String(), jsonMap["accountID"])
	assert.NotNil(t, jsonMap["createdAt"])
}

func TestBankAccountRelatedAccountUnmarshalJSON(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "test",
		Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}

	accountID := models.AccountID{
		Reference:   "acc123",
		ConnectorID: connectorID,
	}

	t.Run("valid JSON", func(t *testing.T) {
		t.Parallel()
		// Given

		encodedAccountID := accountID.String()

		jsonData := `{
			"accountID": "` + encodedAccountID + `",
			"createdAt": "` + now.Format(time.RFC3339Nano) + `"
		}`

		var relatedAccount models.BankAccountRelatedAccount

		err := json.Unmarshal([]byte(jsonData), &relatedAccount)

		// Then
		require.NoError(t, err)

		assert.Equal(t, accountID.String(), relatedAccount.AccountID.String())
		assert.Equal(t, now.Format(time.RFC3339), relatedAccount.CreatedAt.Format(time.RFC3339))
	})

	t.Run("invalid JSON", func(t *testing.T) {
		t.Parallel()
		// Given

		jsonData := `{invalid json}`

		var relatedAccount models.BankAccountRelatedAccount

		err := json.Unmarshal([]byte(jsonData), &relatedAccount)

		// Then
		require.Error(t, err)
	})

	t.Run("invalid accountID", func(t *testing.T) {
		t.Parallel()
		// Given

		jsonData := `{
			"accountID": "invalid-account-id",
			"createdAt": "` + now.Format(time.RFC3339Nano) + `"
		}`

		var relatedAccount models.BankAccountRelatedAccount

		err := json.Unmarshal([]byte(jsonData), &relatedAccount)

		// Then
		require.Error(t, err)
	})
}
