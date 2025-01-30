package models

import (
	"encoding/json"
	"time"
)

type Connector struct {
	// Unique ID of the connector
	ID ConnectorID `json:"id"`
	// Name given by the user to the connector
	Name string `json:"name"`
	// Creation date
	CreatedAt time.Time `json:"createdAt"`
	// Provider type
	Provider string `json:"provider"`
	// ScheduledForDeletion indicates if the connector is scheduled for deletion
	ScheduledForDeletion bool `json:"scheduledForDeletion"`

	// Config given by the user. It will be encrypted when stored
	Config json.RawMessage `json:"config"`
}

func (c *Connector) IdempotencyKey() string {
	return IdempotencyKey(c.ID)
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
		Provider:             c.Provider,
		Config:               c.Config,
		ScheduledForDeletion: c.ScheduledForDeletion,
	})
}

func (c *Connector) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID                   string          `json:"id"`
		Name                 string          `json:"name"`
		CreatedAt            time.Time       `json:"createdAt"`
		Provider             string          `json:"provider"`
		Config               json.RawMessage `json:"config"`
		ScheduledForDeletion bool            `json:"scheduledForDeletion"`
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
	c.Config = aux.Config
	c.ScheduledForDeletion = aux.ScheduledForDeletion

	return nil
}
