// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

import (
	"github.com/formancehq/payments/pkg/client/internal/utils"
	"time"
)

type V3Instance struct {
	ID           string     `json:"id"`
	ConnectorID  string     `json:"connectorID"`
	ScheduleID   string     `json:"scheduleID"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    *time.Time `json:"updatedAt,omitempty"`
	Terminated   bool       `json:"terminated"`
	TerminatedAt *time.Time `json:"terminatedAt,omitempty"`
	Error        *string    `json:"error,omitempty"`
}

func (v V3Instance) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(v, "", false)
}

func (v *V3Instance) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &v, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *V3Instance) GetID() string {
	if o == nil {
		return ""
	}
	return o.ID
}

func (o *V3Instance) GetConnectorID() string {
	if o == nil {
		return ""
	}
	return o.ConnectorID
}

func (o *V3Instance) GetScheduleID() string {
	if o == nil {
		return ""
	}
	return o.ScheduleID
}

func (o *V3Instance) GetCreatedAt() time.Time {
	if o == nil {
		return time.Time{}
	}
	return o.CreatedAt
}

func (o *V3Instance) GetUpdatedAt() *time.Time {
	if o == nil {
		return nil
	}
	return o.UpdatedAt
}

func (o *V3Instance) GetTerminated() bool {
	if o == nil {
		return false
	}
	return o.Terminated
}

func (o *V3Instance) GetTerminatedAt() *time.Time {
	if o == nil {
		return nil
	}
	return o.TerminatedAt
}

func (o *V3Instance) GetError() *string {
	if o == nil {
		return nil
	}
	return o.Error
}
