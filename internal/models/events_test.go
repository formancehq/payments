package models_test

import (
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestEventSent(t *testing.T) {
	t.Parallel()

	t.Run("create and validate EventSent", func(t *testing.T) {
		t.Parallel()
		now := time.Now().UTC()
		connectorID := &models.ConnectorID{
			Provider:  "test",
			Reference: uuid.New(),
		}
		eventID := models.EventID{
			EventIdempotencyKey: "evt123",
		}

		event := models.EventSent{
			ID:          eventID,
			ConnectorID: connectorID,
			SentAt:      now,
		}

		assert.Equal(t, eventID, event.ID)
		assert.Equal(t, connectorID, event.ConnectorID)
		assert.Equal(t, now, event.SentAt)
	})

	t.Run("create EventSent without ConnectorID", func(t *testing.T) {
		t.Parallel()
		now := time.Now().UTC()
		eventID := models.EventID{
			EventIdempotencyKey: "evt123",
		}

		event := models.EventSent{
			ID:     eventID,
			SentAt: now,
		}

		assert.Equal(t, eventID, event.ID)
		assert.Nil(t, event.ConnectorID)
		assert.Equal(t, now, event.SentAt)
	})
}
