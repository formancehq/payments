package models

import (
	"encoding/json"
	"time"
)

type Schedule struct {
	ID           string
	ConnectorID  ConnectorID
	CreatedAt    time.Time
	PausedAt     *time.Time
	PausedReason *string
}

func (s Schedule) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID           string     `json:"id"`
		ConnectorID  string     `json:"connectorID"`
		CreatedAt    time.Time  `json:"createdAt"`
		PausedAt     *time.Time `json:"pausedAt,omitempty"`
		PausedReason *string    `json:"pausedReason,omitempty"`
	}{
		ID:           s.ID,
		ConnectorID:  s.ConnectorID.String(),
		CreatedAt:    s.CreatedAt,
		PausedAt:     s.PausedAt,
		PausedReason: s.PausedReason,
	})
}

func (s *Schedule) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID           string     `json:"id"`
		ConnectorID  string     `json:"connectorID"`
		CreatedAt    time.Time  `json:"createdAt"`
		PausedAt     *time.Time `json:"pausedAt,omitempty"`
		PausedReason *string    `json:"pausedReason,omitempty"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	connectorID, err := ConnectorIDFromString(aux.ConnectorID)
	if err != nil {
		return err
	}

	s.ID = aux.ID
	s.ConnectorID = connectorID
	s.CreatedAt = aux.CreatedAt
	s.PausedAt = aux.PausedAt
	s.PausedReason = aux.PausedReason

	return nil
}
