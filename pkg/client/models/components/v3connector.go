// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

import (
	"github.com/formancehq/payments/pkg/client/internal/utils"
	"time"
)

type Config struct {
}

type V3Connector struct {
	ID                   string    `json:"id"`
	Reference            string    `json:"reference"`
	Name                 string    `json:"name"`
	CreatedAt            time.Time `json:"createdAt"`
	Provider             string    `json:"provider"`
	ScheduledForDeletion bool      `json:"scheduledForDeletion"`
	Config               Config    `json:"config"`
}

func (v V3Connector) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(v, "", false)
}

func (v *V3Connector) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &v, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *V3Connector) GetID() string {
	if o == nil {
		return ""
	}
	return o.ID
}

func (o *V3Connector) GetReference() string {
	if o == nil {
		return ""
	}
	return o.Reference
}

func (o *V3Connector) GetName() string {
	if o == nil {
		return ""
	}
	return o.Name
}

func (o *V3Connector) GetCreatedAt() time.Time {
	if o == nil {
		return time.Time{}
	}
	return o.CreatedAt
}

func (o *V3Connector) GetProvider() string {
	if o == nil {
		return ""
	}
	return o.Provider
}

func (o *V3Connector) GetScheduledForDeletion() bool {
	if o == nil {
		return false
	}
	return o.ScheduledForDeletion
}

func (o *V3Connector) GetConfig() Config {
	if o == nil {
		return Config{}
	}
	return o.Config
}
