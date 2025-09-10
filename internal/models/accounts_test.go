package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToPSPAccount(t *testing.T) {
	t.Parallel()

	t.Run("nil account", func(t *testing.T) {
		t.Parallel()

		// Given
		var account *models.Account = nil

		result := models.ToPSPAccount(account)

		// Then
		assert.Nil(t, result)
	})

	t.Run("non-nil account", func(t *testing.T) {
		t.Parallel()

		// Given
		account := &models.Account{}

		result := models.ToPSPAccount(account)

		// Then
		assert.NotNil(t, result)
	})
}

func TestFromPSPAccount(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	t.Run("valid account", func(t *testing.T) {
		t.Parallel()
		// Given
		pspAccount := models.PSPAccount{
			Reference:    "acc123",
			CreatedAt:    now,
			Name:         pointer.For("Test Account"),
			DefaultAsset: pointer.For("USD/2"),
			Metadata: map[string]string{
				"key": "value",
			},
			Raw: json.RawMessage(`{"test": "data"}`),
		}

		account, err := models.FromPSPAccount(pspAccount, models.ACCOUNT_TYPE_INTERNAL, connectorID)

		// Then
		require.NoError(t, err)
		assert.Equal(t, pspAccount.Reference, account.Reference)
		assert.Equal(t, pspAccount.CreatedAt, account.CreatedAt)
		assert.Equal(t, pspAccount.Name, account.Name)
		assert.Equal(t, pspAccount.DefaultAsset, account.DefaultAsset)
		assert.Equal(t, pspAccount.Metadata, account.Metadata)
		assert.Equal(t, pspAccount.Raw, account.Raw)
		assert.Equal(t, models.ACCOUNT_TYPE_INTERNAL, account.Type)
		assert.Equal(t, connectorID, account.ConnectorID)
	})

	t.Run("validation errors", func(t *testing.T) {
		t.Parallel()

		t.Run("missing reference", func(t *testing.T) {
			t.Parallel()
			// Given
			pspAccount := models.PSPAccount{
				CreatedAt: now,
				Raw:       json.RawMessage(`{}`),
			}

			_, err := models.FromPSPAccount(pspAccount, models.ACCOUNT_TYPE_INTERNAL, connectorID)

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "missing account reference: validation error")
		})

		t.Run("missing createdAt", func(t *testing.T) {
			t.Parallel()
			// Given
			pspAccount := models.PSPAccount{
				Reference: "acc123",
				Raw:       json.RawMessage(`{}`),
			}

			_, err := models.FromPSPAccount(pspAccount, models.ACCOUNT_TYPE_INTERNAL, connectorID)

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "missing account createdAt: validation error")
		})

		t.Run("missing raw", func(t *testing.T) {
			t.Parallel()
			// Given
			pspAccount := models.PSPAccount{
				Reference: "acc123",
				CreatedAt: now,
			}

			_, err := models.FromPSPAccount(pspAccount, models.ACCOUNT_TYPE_INTERNAL, connectorID)

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "missing account raw: validation error")
		})

		t.Run("invalid defaultAsset", func(t *testing.T) {
			t.Parallel()
			// Given
			pspAccount := models.PSPAccount{
				Reference:    "acc123",
				CreatedAt:    now,
				DefaultAsset: pointer.For("invalid"),
				Raw:          json.RawMessage(`{}`),
			}

			_, err := models.FromPSPAccount(pspAccount, models.ACCOUNT_TYPE_INTERNAL, connectorID)

			// Then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid default asset: validation error")
		})
	})
}

func TestFromPSPAccounts(t *testing.T) {
	t.Parallel()

	// Given
	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	pspAccounts := []models.PSPAccount{
		{
			Reference: "acc1",
			CreatedAt: now,
			Raw:       json.RawMessage(`{}`),
		},
		{
			Reference: "acc2",
			CreatedAt: now,
			Raw:       json.RawMessage(`{}`),
		},
	}

	accounts, err := models.FromPSPAccounts(pspAccounts, models.ACCOUNT_TYPE_INTERNAL, connectorID)

	// Then
	require.NoError(t, err)
	assert.Len(t, accounts, 2)
	assert.Equal(t, "acc1", accounts[0].Reference)
	assert.Equal(t, "acc2", accounts[1].Reference)

	// Given
	invalidPspAccounts := append(pspAccounts, models.PSPAccount{
		CreatedAt: now,
		Raw:       json.RawMessage(`{}`),
	})

	_, err = models.FromPSPAccounts(invalidPspAccounts, models.ACCOUNT_TYPE_INTERNAL, connectorID)

	// Then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing account reference: validation error")
}

func TestAccountMarshalUnmarshal(t *testing.T) {
	t.Parallel()

	// Given
	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	account := models.Account{
		ID: models.AccountID{
			Reference:   "acc123",
			ConnectorID: connectorID,
		},
		ConnectorID:  connectorID,
		Reference:    "acc123",
		CreatedAt:    now,
		Type:         models.ACCOUNT_TYPE_INTERNAL,
		Name:         pointer.For("Test Account"),
		DefaultAsset: pointer.For("USD/2"),
		Metadata: map[string]string{
			"key": "value",
		},
		Raw: json.RawMessage(`{"test": "data"}`),
	}

	data, err := json.Marshal(account)

	// Then
	require.NoError(t, err)

	var unmarshaledAccount models.Account
	err = json.Unmarshal(data, &unmarshaledAccount)

	// Then
	require.NoError(t, err)

	assert.Equal(t, account.ID, unmarshaledAccount.ID)
	assert.Equal(t, account.ConnectorID, unmarshaledAccount.ConnectorID)
	assert.Equal(t, account.Reference, unmarshaledAccount.Reference)
	assert.Equal(t, account.CreatedAt, unmarshaledAccount.CreatedAt)
	assert.Equal(t, account.Type, unmarshaledAccount.Type)
	assert.Equal(t, account.Name, unmarshaledAccount.Name)
	assert.Equal(t, account.DefaultAsset, unmarshaledAccount.DefaultAsset)
	assert.Equal(t, account.Metadata, unmarshaledAccount.Metadata)

	var originalJSON, unmarshaledJSON interface{}
	err = json.Unmarshal(account.Raw, &originalJSON)

	// Then
	require.NoError(t, err)

	err = json.Unmarshal(unmarshaledAccount.Raw, &unmarshaledJSON)

	// Then
	require.NoError(t, err)
	assert.Equal(t, originalJSON, unmarshaledJSON)
}

func TestAccountIdempotencyKey(t *testing.T) {
	t.Parallel()

	// Given
	connectorID := models.ConnectorID{
		Reference: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Provider:  "test",
	}

	account := models.Account{
		ID: models.AccountID{
			Reference:   "acc123",
			ConnectorID: connectorID,
		},
	}

	key := account.IdempotencyKey()

	// Then
	assert.NotEmpty(t, key)
	assert.Regexp(t, "^[0-9a-f]{40}$", key)
}

func TestPSPAccountValidate(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	t.Run("valid account", func(t *testing.T) {
		t.Parallel()

		// Given
		account := models.PSPAccount{
			Reference: "acc123",
			CreatedAt: now,
			Raw:       json.RawMessage(`{}`),
		}

		err := account.Validate()

		// Then
		assert.NoError(t, err)
	})

	t.Run("missing reference", func(t *testing.T) {
		t.Parallel()

		// Given
		account := models.PSPAccount{
			CreatedAt: now,
			Raw:       json.RawMessage(`{}`),
		}

		err := account.Validate()

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing account reference: validation error")
	})

	t.Run("missing createdAt", func(t *testing.T) {
		t.Parallel()

		// Given
		account := models.PSPAccount{
			Reference: "acc123",
			Raw:       json.RawMessage(`{}`),
		}

		err := account.Validate()

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing account createdAt: validation error")
	})

	t.Run("missing raw", func(t *testing.T) {
		t.Parallel()

		// Given
		account := models.PSPAccount{
			Reference: "acc123",
			CreatedAt: now,
		}

		err := account.Validate()

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing account raw: validation error")
	})

	t.Run("invalid defaultAsset", func(t *testing.T) {
		t.Parallel()

		// Given
		account := models.PSPAccount{
			Reference:    "acc123",
			CreatedAt:    now,
			DefaultAsset: pointer.For("invalid"),
			Raw:          json.RawMessage(`{}`),
		}

		err := account.Validate()

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid default asset: validation error")
	})
}
