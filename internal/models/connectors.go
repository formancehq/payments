package models

import (
	"encoding/json"
	"time"
)

// simplified view without private info
type ConnectorBase struct {
	// Unique ID of the connector
	ID ConnectorID `json:"id"`
	// Name given by the user to the connector
	Name string `json:"name"`
	// Creation date
	CreatedAt time.Time `json:"createdAt"`
	// Provider type
	Provider string `json:"provider"`
}

func (c ConnectorBase) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID        string    `json:"id"`
		Reference string    `json:"reference"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"createdAt"`
		Provider  string    `json:"provider"`
	}{
		ID:        c.ID.String(),
		Reference: c.ID.Reference.String(),
		Name:      c.Name,
		CreatedAt: c.CreatedAt,
		Provider:  ToV3Provider(c.Provider),
	})
}

func (c *ConnectorBase) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID        string    `json:"id"`
		Reference string    `json:"reference"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"createdAt"`
		Provider  string    `json:"provider"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	id, err := ConnectorIDFromString(aux.ID)
	if err != nil {
		return err
	}

	c.ID = id
	c.Name = aux.Name
	c.CreatedAt = aux.CreatedAt
	c.Provider = aux.Provider
	return nil
}

type Connector struct {
	ConnectorBase

	// ScheduledForDeletion indicates if the connector is scheduled for deletion
	ScheduledForDeletion bool `json:"scheduledForDeletion"`

	// Config given by the user. It will be encrypted when stored
	Config json.RawMessage `json:"config"`
}

func (c *Connector) IdempotencyKey() string {
	return IdempotencyKey(c.ID)
}

func (c *Connector) Base() *ConnectorBase {
	return &ConnectorBase{
		ID:        c.ID,
		Name:      c.Name,
		CreatedAt: c.CreatedAt,
		Provider:  c.Provider,
	}
}

func (c Connector) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID                   string          `json:"id"`
		Reference            string          `json:"reference"`
		Name                 string          `json:"name"`
		CreatedAt            time.Time       `json:"createdAt"`
		Provider             string          `json:"provider"`
		Config               json.RawMessage `json:"config"`
		ScheduledForDeletion bool            `json:"scheduledForDeletion"`
	}{
		ID:                   c.ID.String(),
		Reference:            c.ID.Reference.String(),
		Name:                 c.Name,
		CreatedAt:            c.CreatedAt,
		Provider:             ToV3Provider(c.Provider),
		Config:               c.Config,
		ScheduledForDeletion: c.ScheduledForDeletion,
	})
}

func (c *Connector) UnmarshalJSON(data []byte) error {
	var aux struct {
		Config               json.RawMessage `json:"config"`
		ScheduledForDeletion bool            `json:"scheduledForDeletion"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var base ConnectorBase
	if err := json.Unmarshal(data, &base); err != nil {
		return err
	}

	c.ID = base.ID
	c.Name = base.Name
	c.CreatedAt = base.CreatedAt
	c.Provider = base.Provider
	c.Config = aux.Config
	c.ScheduledForDeletion = aux.ScheduledForDeletion

	return nil
}

// When using the provider inside the connectorID struct, we need to convert it
// to the v3 version. This is because we can't change the connectorID struct
// when migrating from v2 to v3 because we do not want to break the API for the
// client if they are relying on the connectorID on their side.
func ToV3Provider(provider string) string {
	switch provider {
	case "ADYEN":
		return "adyen"
	case "ATLAR":
		return "atlar"
	case "BANKING-CIRCLE":
		return "bankingcircle"
	case "CURRENCY-CLOUD":
		return "currencycloud"
	case "DUMMY-PAY":
		return "dummypay"
	case "GENERIC":
		return "generic"
	case "MANGOPAY":
		return "mangopay"
	case "MODULR":
		return "modulr"
	case "MONEYCORP":
		return "moneycorp"
	case "STRIPE":
		return "stripe"
	case "WISE":
		return "wise"
	default:
		return provider
	}
}
