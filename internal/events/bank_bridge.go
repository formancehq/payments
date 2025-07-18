package events

import (
	"time"

	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
	"github.com/google/uuid"
)

type BankBridgeUserLinkStatus struct {
	PsuID       uuid.UUID `json:"psuID"`
	ConnectorID string    `json:"connectorID"`
	Status      string    `json:"status"`
	Error       *string   `json:"error,omitempty"`
}

func (e Events) NewEventBankBridgeUserLinkStatus(userLinkStatus models.UserLinkSessionFinished) publish.EventMessage {
	payload := BankBridgeUserLinkStatus{
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
		Type:           events.EventTypeBankBridgeUserLinkStatus,
		Payload:        payload,
	}
}

type BankBridgeUserConnectionDataSynced struct {
	PsuID        uuid.UUID `json:"psuID"`
	ConnectorID  string    `json:"connectorID"`
	ConnectionID string    `json:"connectionID"`
	At           time.Time `json:"at"`
}

func (e Events) NewEventBankBridgeUserConnectionDataSynced(userConnectionUpdated models.UserConnectionDataSynced) publish.EventMessage {
	payload := BankBridgeUserConnectionDataSynced{
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
		Type:           events.EventTypeBankBridgeUserConnectionDataSynced,
		Payload:        payload,
	}
}

type BankBridgeUserConnectionPendingDisconnect struct {
	PsuID        uuid.UUID `json:"psuID"`
	ConnectorID  string    `json:"connectorID"`
	ConnectionID string    `json:"connectionID"`
	At           time.Time `json:"at"`
	Reason       *string   `json:"reason"`
}

func (e Events) NewEventBankBridgeUserPendingDisconnect(userPendingDisconnect models.UserConnectionPendingDisconnect) publish.EventMessage {
	payload := BankBridgeUserConnectionPendingDisconnect{
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
		Type:           events.EventTypeBankBridgeUserConnectionPendingDisconnect,
		Payload:        payload,
	}
}

type BankBridgeUserConnectionDisconnected struct {
	PsuID        uuid.UUID `json:"psuID"`
	ConnectorID  string    `json:"connectorID"`
	ConnectionID string    `json:"connectionID"`
	At           time.Time `json:"at"`
	Reason       *string   `json:"reason"`
}

func (e Events) NewEventBankBridgeUserDisconnected(userDisconnected models.UserConnectionDisconnected) publish.EventMessage {
	payload := BankBridgeUserConnectionDisconnected{
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
		Type:           events.EventTypeBankBridgeUserConnectionDisconnected,
		Payload:        payload,
	}
}
