package models_test

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookStructs(t *testing.T) {
	t.Parallel()

	t.Run("PSPWebhookConfig", func(t *testing.T) {
		t.Parallel()
		config := models.PSPWebhookConfig{
			Name:    "test-webhook",
			URLPath: "/webhooks/test",
		}

		data, err := json.Marshal(config)

		require.NoError(t, err)
		var unmarshaled models.PSPWebhookConfig
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, config.Name, unmarshaled.Name)
		assert.Equal(t, config.URLPath, unmarshaled.URLPath)
	})

	t.Run("WebhookConfig", func(t *testing.T) {
		t.Parallel()
		connectorID := models.ConnectorID{
			Provider:  "test",
			Reference: uuid.New(),
		}
		config := models.WebhookConfig{
			Name:        "test-webhook",
			ConnectorID: connectorID,
			URLPath:     "/webhooks/test",
		}

		data, err := json.Marshal(config)

		require.NoError(t, err)
		var unmarshaled models.WebhookConfig
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, config.Name, unmarshaled.Name)
		assert.Equal(t, config.ConnectorID, unmarshaled.ConnectorID)
		assert.Equal(t, config.URLPath, unmarshaled.URLPath)
	})

	t.Run("BasicAuth", func(t *testing.T) {
		t.Parallel()
		auth := models.BasicAuth{
			Username: "user",
			Password: "pass",
		}

		data, err := json.Marshal(auth)

		require.NoError(t, err)
		var unmarshaled models.BasicAuth
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, auth.Username, unmarshaled.Username)
		assert.Equal(t, auth.Password, unmarshaled.Password)
	})

	t.Run("PSPWebhook", func(t *testing.T) {
		t.Parallel()
		auth := &models.BasicAuth{
			Username: "user",
			Password: "pass",
		}
		webhook := models.PSPWebhook{
			BasicAuth: auth,
			QueryValues: map[string][]string{
				"key": {"value1", "value2"},
			},
			Headers: map[string][]string{
				"Content-Type": {"application/json"},
			},
			Body: []byte(`{"test": "data"}`),
		}

		data, err := json.Marshal(webhook)

		require.NoError(t, err)
		var unmarshaled models.PSPWebhook
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, webhook.BasicAuth.Username, unmarshaled.BasicAuth.Username)
		assert.Equal(t, webhook.BasicAuth.Password, unmarshaled.BasicAuth.Password)
		assert.Equal(t, webhook.QueryValues, unmarshaled.QueryValues)
		assert.Equal(t, webhook.Headers, unmarshaled.Headers)
		assert.Equal(t, webhook.Body, unmarshaled.Body)
	})

	t.Run("Webhook", func(t *testing.T) {
		t.Parallel()
		connectorID := models.ConnectorID{
			Provider:  "test",
			Reference: uuid.New(),
		}
		auth := &models.BasicAuth{
			Username: "user",
			Password: "pass",
		}
		webhook := models.Webhook{
			ID:          "webhook123",
			ConnectorID: connectorID,
			BasicAuth:   auth,
			QueryValues: map[string][]string{
				"key": {"value1", "value2"},
			},
			Headers: map[string][]string{
				"Content-Type": {"application/json"},
			},
			Body: []byte(`{"test": "data"}`),
		}

		data, err := json.Marshal(webhook)

		require.NoError(t, err)
		var unmarshaled models.Webhook
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, webhook.ID, unmarshaled.ID)
		assert.Equal(t, webhook.ConnectorID, unmarshaled.ConnectorID)
		assert.Equal(t, webhook.BasicAuth.Username, unmarshaled.BasicAuth.Username)
		assert.Equal(t, webhook.BasicAuth.Password, unmarshaled.BasicAuth.Password)
		assert.Equal(t, webhook.QueryValues, unmarshaled.QueryValues)
		assert.Equal(t, webhook.Headers, unmarshaled.Headers)
		assert.Equal(t, webhook.Body, unmarshaled.Body)
	})

	t.Run("PSPWebhook without BasicAuth", func(t *testing.T) {
		t.Parallel()
		webhook := models.PSPWebhook{
			QueryValues: map[string][]string{
				"key": {"value"},
			},
			Headers: map[string][]string{
				"Content-Type": {"application/json"},
			},
			Body: []byte(`{"test": "data"}`),
		}

		data, err := json.Marshal(webhook)

		require.NoError(t, err)
		var unmarshaled models.PSPWebhook
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		assert.Nil(t, unmarshaled.BasicAuth)
		assert.Equal(t, webhook.QueryValues, unmarshaled.QueryValues)
		assert.Equal(t, webhook.Headers, unmarshaled.Headers)
		assert.Equal(t, webhook.Body, unmarshaled.Body)
	})
}
