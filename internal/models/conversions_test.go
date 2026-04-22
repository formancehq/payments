package models_test

import (
	"encoding/json"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validPSPConversion() models.PSPConversion {
	return models.PSPConversion{
		Reference:         "conv-ref-1",
		CreatedAt:         time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		SourceAsset:       "USD/2",
		DestinationAsset:  "USDC/6",
		SourceAmount:      big.NewInt(1000),
		DestinationAmount: big.NewInt(999),
		Fee:               big.NewInt(1),
		FeeAsset:          pointer.For("USD/2"),
		Status:            models.CONVERSION_STATUS_COMPLETED,
		SourceAccountReference:      pointer.For("src-wallet"),
		DestinationAccountReference: pointer.For("dst-wallet"),
		Metadata:                    map[string]string{"k": "v"},
		Raw:                         json.RawMessage(`{"raw":"ok"}`),
	}
}

func TestPSPConversionValidate(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		c := validPSPConversion()
		assert.NoError(t, c.Validate())
	})

	cases := []struct {
		name   string
		mutate func(*models.PSPConversion)
		errMsg string
	}{
		{"missing reference", func(c *models.PSPConversion) { c.Reference = "" }, "missing conversion reference"},
		{"missing createdAt", func(c *models.PSPConversion) { c.CreatedAt = time.Time{} }, "missing conversion createdAt"},
		{"invalid source asset", func(c *models.PSPConversion) { c.SourceAsset = "nope" }, "invalid conversion source asset"},
		{"invalid destination asset", func(c *models.PSPConversion) { c.DestinationAsset = "nope" }, "invalid conversion destination asset"},
		{"missing source amount", func(c *models.PSPConversion) { c.SourceAmount = nil }, "missing conversion source amount"},
		{"missing status", func(c *models.PSPConversion) { c.Status = models.CONVERSION_STATUS_UNKNOWN }, "missing conversion status"},
		{"missing raw", func(c *models.PSPConversion) { c.Raw = nil }, "missing conversion raw"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := validPSPConversion()
			tc.mutate(&c)
			err := c.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errMsg)
		})
	}
}

func TestFromPSPConversionToConversion(t *testing.T) {
	t.Parallel()
	connectorID := newConnectorID(t)

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		psp := validPSPConversion()
		conv, err := models.FromPSPConversionToConversion(psp, connectorID)
		require.NoError(t, err)

		assert.Equal(t, psp.Reference, conv.Reference)
		assert.Equal(t, psp.CreatedAt, conv.CreatedAt)
		assert.Equal(t, psp.SourceAsset, conv.SourceAsset)
		assert.Equal(t, psp.DestinationAsset, conv.DestinationAsset)
		assert.Equal(t, psp.SourceAmount, conv.SourceAmount)
		assert.Equal(t, psp.DestinationAmount, conv.DestinationAmount)
		assert.Equal(t, psp.Status, conv.Status)
		assert.Equal(t, connectorID, conv.ConnectorID)

		require.NotNil(t, conv.SourceAccountID)
		require.NotNil(t, conv.DestinationAccountID)
		assert.Equal(t, "src-wallet", conv.SourceAccountID.Reference)
		assert.Equal(t, "dst-wallet", conv.DestinationAccountID.Reference)

		// UpdatedAt stamped to "now" – just make sure it's non-zero.
		assert.False(t, conv.UpdatedAt.IsZero())

		// IdempotencyKey is deterministic on (ID, Status).
		assert.NotEmpty(t, conv.IdempotencyKey())
	})

	t.Run("nil account refs -> nil account ids", func(t *testing.T) {
		t.Parallel()
		psp := validPSPConversion()
		psp.SourceAccountReference = nil
		psp.DestinationAccountReference = nil

		conv, err := models.FromPSPConversionToConversion(psp, connectorID)
		require.NoError(t, err)
		assert.Nil(t, conv.SourceAccountID)
		assert.Nil(t, conv.DestinationAccountID)
	})

	t.Run("invalid conversion returns error", func(t *testing.T) {
		t.Parallel()
		psp := validPSPConversion()
		psp.Reference = ""
		_, err := models.FromPSPConversionToConversion(psp, connectorID)
		require.Error(t, err)
	})
}

func TestFromPSPConversions(t *testing.T) {
	t.Parallel()
	connectorID := newConnectorID(t)

	t.Run("all valid", func(t *testing.T) {
		t.Parallel()
		a := validPSPConversion()
		a.Reference = "a"
		b := validPSPConversion()
		b.Reference = "b"

		convs, err := models.FromPSPConversions([]models.PSPConversion{a, b}, connectorID)
		require.NoError(t, err)
		require.Len(t, convs, 2)
		assert.Equal(t, "a", convs[0].Reference)
		assert.Equal(t, "b", convs[1].Reference)
	})

	t.Run("empty slice", func(t *testing.T) {
		t.Parallel()
		convs, err := models.FromPSPConversions(nil, connectorID)
		require.NoError(t, err)
		assert.Empty(t, convs)
	})

	t.Run("invalid aborts", func(t *testing.T) {
		t.Parallel()
		bad := validPSPConversion()
		bad.Reference = ""
		_, err := models.FromPSPConversions([]models.PSPConversion{bad}, connectorID)
		require.Error(t, err)
	})
}

func TestConversionMarshalUnmarshal(t *testing.T) {
	t.Parallel()
	connectorID := newConnectorID(t)

	conv, err := models.FromPSPConversionToConversion(validPSPConversion(), connectorID)
	require.NoError(t, err)

	data, err := json.Marshal(conv)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"provider":`)

	var decoded models.Conversion
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, conv.Reference, decoded.Reference)
	assert.Equal(t, conv.SourceAsset, decoded.SourceAsset)
	assert.Equal(t, conv.DestinationAsset, decoded.DestinationAsset)
	assert.Equal(t, conv.SourceAmount, decoded.SourceAmount)
	assert.Equal(t, conv.Status, decoded.Status)
	assert.Equal(t, conv.ConnectorID, decoded.ConnectorID)
	require.NotNil(t, decoded.SourceAccountID)
	assert.Equal(t, conv.SourceAccountID.Reference, decoded.SourceAccountID.Reference)
}

func TestConversionUnmarshalInvalid(t *testing.T) {
	t.Parallel()
	connectorID := newConnectorID(t)
	conv, err := models.FromPSPConversionToConversion(validPSPConversion(), connectorID)
	require.NoError(t, err)

	t.Run("malformed JSON", func(t *testing.T) {
		t.Parallel()
		var c models.Conversion
		err := json.Unmarshal([]byte("not json"), &c)
		require.Error(t, err)
	})

	t.Run("invalid id", func(t *testing.T) {
		t.Parallel()
		var c models.Conversion
		err := json.Unmarshal([]byte(`{"id":"!!!"}`), &c)
		require.Error(t, err)
	})

	t.Run("invalid connector id", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{
			"id":          conv.ID.String(),
			"connectorID": "not-a-connector-id",
		}
		raw, err := json.Marshal(payload)
		require.NoError(t, err)

		var c models.Conversion
		err = json.Unmarshal(raw, &c)
		require.Error(t, err)
	})

	t.Run("invalid source account id", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{
			"id":              conv.ID.String(),
			"connectorID":     connectorID.String(),
			"sourceAccountID": "invalid-account-id",
		}
		raw, err := json.Marshal(payload)
		require.NoError(t, err)

		var c models.Conversion
		err = json.Unmarshal(raw, &c)
		require.Error(t, err)
	})

	t.Run("invalid destination account id", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{
			"id":                   conv.ID.String(),
			"connectorID":          connectorID.String(),
			"destinationAccountID": "invalid-account-id",
		}
		raw, err := json.Marshal(payload)
		require.NoError(t, err)

		var c models.Conversion
		err = json.Unmarshal(raw, &c)
		require.Error(t, err)
	})
}

func TestConversionExpandedMarshal(t *testing.T) {
	t.Parallel()
	connectorID := newConnectorID(t)
	conv, err := models.FromPSPConversionToConversion(validPSPConversion(), connectorID)
	require.NoError(t, err)

	t.Run("without error", func(t *testing.T) {
		t.Parallel()
		ce := models.ConversionExpanded{Conversion: conv, Status: models.CONVERSION_STATUS_COMPLETED}
		data, err := json.Marshal(ce)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"status":"COMPLETED"`)
		assert.NotContains(t, string(data), `"error"`)
	})

	t.Run("with error", func(t *testing.T) {
		t.Parallel()
		ce := models.ConversionExpanded{
			Conversion: conv,
			Status:     models.CONVERSION_STATUS_FAILED,
			Error:      errors.New("kaboom"),
		}
		data, err := json.Marshal(ce)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"error":"kaboom"`)
	})
}
