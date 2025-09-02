package events

import (
	"time"

	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
	"github.com/google/uuid"
)

type OpenBankingUserLinkStatus struct {
	PsuID       uuid.UUID `json:"psuID"`
	ConnectorID string    `json:"connectorID"`
	Status      string    `json:"status"`
	Error       *string   `json:"error,omitempty"`
}

func (e Events) NewEventOpenBankingUserLinkStatus(userLinkStatus models.UserLinkSessionFinished) publish.EventMessage {
	payload := OpenBankingUserLinkStatus{
		PsuID:       userLinkStatus.PsuID,
		ConnectorID: userLinkStatus.ConnectorID.String(),
		Status:      string(userLinkStatus.Status),
		Error:       userLinkStatus.Error,
	}

	ik := models.IdempotencyKey(payload)

	return publish.EventMessage{
		IdempotencyKey: ik,
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeOpenBankingUserLinkStatus,
		Payload:        payload,
	}
}

type OpenBankingUserConnectionDataSynced struct {
	PsuID        uuid.UUID `json:"psuID"`
	ConnectorID  string    `json:"connectorID"`
	ConnectionID string    `json:"connectionID"`
	At           time.Time `json:"at"`
}

func (e Events) NewEventOpenBankingUserConnectionDataSynced(userConnectionUpdated models.UserConnectionDataSynced) publish.EventMessage {
	payload := OpenBankingUserConnectionDataSynced{
		PsuID:        userConnectionUpdated.PsuID,
		ConnectorID:  userConnectionUpdated.ConnectorID.String(),
		ConnectionID: userConnectionUpdated.ConnectionID,
		At:           userConnectionUpdated.At,
	}

	ik := models.IdempotencyKey(payload)

	return publish.EventMessage{
		IdempotencyKey: ik,
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeOpenBankingUserConnectionDataSynced,
		Payload:        payload,
	}
}

type OpenBankingUserConnectionPendingDisconnect struct {
	PsuID        uuid.UUID `json:"psuID"`
	ConnectorID  string    `json:"connectorID"`
	ConnectionID string    `json:"connectionID"`
	At           time.Time `json:"at"`
	Reason       *string   `json:"reason"`
}

func (e Events) NewEventOpenBankingUserPendingDisconnect(userPendingDisconnect models.UserConnectionPendingDisconnect) publish.EventMessage {
	payload := OpenBankingUserConnectionPendingDisconnect{
		PsuID:        userPendingDisconnect.PsuID,
		ConnectorID:  userPendingDisconnect.ConnectorID.String(),
		ConnectionID: userPendingDisconnect.ConnectionID,
		At:           userPendingDisconnect.At,
		Reason:       userPendingDisconnect.Reason,
	}

	ik := models.IdempotencyKey(payload)

	return publish.EventMessage{
		IdempotencyKey: ik,
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeOpenBankingUserConnectionPendingDisconnect,
		Payload:        payload,
	}
}

type OpenBankingUserConnectionDisconnected struct {
	PsuID        uuid.UUID `json:"psuID"`
	ConnectorID  string    `json:"connectorID"`
	ConnectionID string    `json:"connectionID"`
	At           time.Time `json:"at"`
	Reason       *string   `json:"reason,omitempty"`
}

func (e Events) NewEventOpenBankingUserConnectionDisconnected(userDisconnected models.UserConnectionDisconnected) publish.EventMessage {
	payload := OpenBankingUserConnectionDisconnected{
		PsuID:        userDisconnected.PsuID,
		ConnectorID:  userDisconnected.ConnectorID.String(),
		ConnectionID: userDisconnected.ConnectionID,
		At:           userDisconnected.At,
		Reason:       userDisconnected.Reason,
	}

	ik := models.IdempotencyKey(payload)

	return publish.EventMessage{
		IdempotencyKey: ik,
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeOpenBankingUserConnectionDisconnected,
		Payload:        payload,
	}
}

type OpenBankingUserConnectionReconnected struct {
	PsuID        uuid.UUID `json:"psuID"`
	ConnectorID  string    `json:"connectorID"`
	ConnectionID string    `json:"connectionID"`
	At           time.Time `json:"at"`
}

func (e Events) NewEventOpenBankingUserConnectionReconnected(userReconnected models.UserConnectionReconnected) publish.EventMessage {
	payload := OpenBankingUserConnectionReconnected{
		PsuID:        userReconnected.PsuID,
		ConnectorID:  userReconnected.ConnectorID.String(),
		ConnectionID: userReconnected.ConnectionID,
		At:           userReconnected.At,
	}

	ik := models.IdempotencyKey(payload)

	return publish.EventMessage{
		IdempotencyKey: ik,
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeOpenBankingUserConnectionReconnected,
		Payload:        payload,
	}
}

type OpenBankingUserDisconnected struct {
	PsuID       uuid.UUID `json:"psuID"`
	ConnectorID string    `json:"connectorID"`
	At          time.Time `json:"at"`
	Reason      *string   `json:"reason,omitempty"`
}

func (e Events) NewEventOpenBankingUserDisconnected(userDisconnected models.UserDisconnected) publish.EventMessage {
	payload := OpenBankingUserDisconnected{
		PsuID:       userDisconnected.PsuID,
		ConnectorID: userDisconnected.ConnectorID.String(),
		At:          userDisconnected.At,
		Reason:      userDisconnected.Reason,
	}

	return publish.EventMessage{
		IdempotencyKey: models.IdempotencyKey(payload),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeOpenBankingUserDisconnected,
		Payload:        payload,
	}
}
