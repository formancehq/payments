// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

import (
	"github.com/formancehq/payments/pkg/client/internal/utils"
	"time"
)

type V3Pool struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"createdAt"`
	PoolAccounts []string  `json:"poolAccounts"`
}

func (v V3Pool) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(v, "", false)
}

func (v *V3Pool) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &v, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *V3Pool) GetID() string {
	if o == nil {
		return ""
	}
	return o.ID
}

func (o *V3Pool) GetName() string {
	if o == nil {
		return ""
	}
	return o.Name
}

func (o *V3Pool) GetCreatedAt() time.Time {
	if o == nil {
		return time.Time{}
	}
	return o.CreatedAt
}

func (o *V3Pool) GetPoolAccounts() []string {
	if o == nil {
		return []string{}
	}
	return o.PoolAccounts
}
