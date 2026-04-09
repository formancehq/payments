package models

import (
	"encoding/json"
	"time"
)

type Instance struct {
	ID           string
	ScheduleID   string
	ConnectorID  ConnectorID
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Terminated   bool
	TerminatedAt *time.Time
	Error        *string
}

func (i Instance) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID           string     `json:"id"`
		ScheduleID   string     `json:"scheduleID"`
		ConnectorID  string     `json:"connectorID"`
		CreatedAt    time.Time  `json:"createdAt"`
		UpdatedAt    time.Time  `json:"updatedAt"`
		Terminated   bool       `json:"terminated"`
		TerminatedAt *time.Time `json:"terminatedAt,omitempty"`
		Error        *string    `json:"error,omitempty"`
	}{
		ID:           i.ID,
		ScheduleID:   i.ScheduleID,
		ConnectorID:  i.ConnectorID.String(),
		CreatedAt:    i.CreatedAt,
		UpdatedAt:    i.UpdatedAt,
		Terminated:   i.Terminated,
		TerminatedAt: i.TerminatedAt,
		Error:        i.Error,
	})
}

func (i *Instance) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID           string     `json:"id"`
		ScheduleID   string     `json:"scheduleID"`
		ConnectorID  string     `json:"connectorID"`
		CreatedAt    time.Time  `json:"createdAt"`
		UpdatedAt    time.Time  `json:"updatedAt"`
		Terminated   bool       `json:"terminated"`
		TerminatedAt *time.Time `json:"terminatedAt,omitempty"`
		Error        *string    `json:"error,omitempty"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	connectorID, err := ConnectorIDFromString(aux.ConnectorID)
	if err != nil {
		return err
	}

	i.ID = aux.ID
	i.ConnectorID = connectorID
	i.ScheduleID = aux.ScheduleID
	i.CreatedAt = aux.CreatedAt
	i.UpdatedAt = aux.UpdatedAt
	i.Terminated = aux.Terminated
	i.TerminatedAt = aux.TerminatedAt
	i.Error = aux.Error

	return nil
}
