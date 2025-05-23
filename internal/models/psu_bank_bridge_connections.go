package models

import (
	"github.com/google/uuid"
)

type PSUBankBridgeConnectionAttempt struct {
	ID           uuid.UUID   `json:"id"`
	ConnectionID ConnectorID `json:"connectionID"`

	// Optional
	TemporaryToken *string        `json:"temporaryToken"`
	State          *CallbackState `json:"state"`
}

type PSUBankBridgeConnections struct {
	ID uuid.UUID `json:"id"`

	ConnectorID ConnectorID `json:"connectorID"`

	// Optional
	// AuthToken is optional for some banking bridges, like Powens, where we
	// have a notion of connection, but we only have one token for all of them.
	AuthToken *string `json:"authToken"`
	// per banking bridge additional information
	Metadata map[string]string `json:"metadata"`

	Connections []PSUBankBridgeConnection `json:"connections"`
}

type PSUBankBridgeConnection struct {
	ConnectionID string `json:"connectionID"`

	// Optional
	// AccessToken is optional for some banking bridges, like Powens, where we
	// have a notion of connection, but we only have one token for all of them.
	AccessToken *string `json:"accessToken"`
	// Additional information about the connection depending on the connector
	AdditionalInformation map[string]string `json:"metadata"`
}
