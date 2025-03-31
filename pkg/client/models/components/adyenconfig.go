// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

import (
	"github.com/formancehq/payments/pkg/client/internal/utils"
)

type AdyenConfig struct {
	Name               string  `json:"name"`
	Provider           *string `default:"Adyen" json:"provider"`
	APIKey             string  `json:"apiKey"`
	HmacKey            string  `json:"hmacKey"`
	LiveEndpointPrefix *string `json:"liveEndpointPrefix,omitempty"`
	// The frequency at which the connector will try to fetch new BalanceTransaction objects from Adyen API.
	//
	PollingPeriod *string `default:"120s" json:"pollingPeriod"`
}

func (a AdyenConfig) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(a, "", false)
}

func (a *AdyenConfig) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &a, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *AdyenConfig) GetName() string {
	if o == nil {
		return ""
	}
	return o.Name
}

func (o *AdyenConfig) GetProvider() *string {
	if o == nil {
		return nil
	}
	return o.Provider
}

func (o *AdyenConfig) GetAPIKey() string {
	if o == nil {
		return ""
	}
	return o.APIKey
}

func (o *AdyenConfig) GetHmacKey() string {
	if o == nil {
		return ""
	}
	return o.HmacKey
}

func (o *AdyenConfig) GetLiveEndpointPrefix() *string {
	if o == nil {
		return nil
	}
	return o.LiveEndpointPrefix
}

func (o *AdyenConfig) GetPollingPeriod() *string {
	if o == nil {
		return nil
	}
	return o.PollingPeriod
}
