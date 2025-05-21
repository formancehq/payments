package models

import "github.com/google/uuid"

type PSUBankBridgeConsent struct {
	ID uuid.UUID `json:"id"`

	ConnectorID uuid.UUID `json:"connectorID"`
	AccessToken string    `json:"accessToken"`

	Connections []PSUBankBridgeConnection `json:"connections"`
}

type PSUBankBridgeConnection struct {
	ConnectionID string `json:"connectionID"`

	// Additional information about the connection depending on the connector
	AdditionalInformation map[string]string `json:"metadata"`
}
