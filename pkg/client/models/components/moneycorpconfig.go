// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

import (
	"github.com/formancehq/payments/pkg/client/internal/utils"
)

type MoneycorpConfig struct {
	Name     string  `json:"name"`
	Provider *string `default:"Moneycorp" json:"provider"`
	ClientID string  `json:"clientID"`
	APIKey   string  `json:"apiKey"`
	Endpoint string  `json:"endpoint"`
	// The frequency at which the connector will try to fetch new BalanceTransaction objects from MoneyCorp API.
	//
	PollingPeriod *string `default:"120s" json:"pollingPeriod"`
}

func (m MoneycorpConfig) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(m, "", false)
}

func (m *MoneycorpConfig) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &m, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *MoneycorpConfig) GetName() string {
	if o == nil {
		return ""
	}
	return o.Name
}

func (o *MoneycorpConfig) GetProvider() *string {
	if o == nil {
		return nil
	}
	return o.Provider
}

func (o *MoneycorpConfig) GetClientID() string {
	if o == nil {
		return ""
	}
	return o.ClientID
}

func (o *MoneycorpConfig) GetAPIKey() string {
	if o == nil {
		return ""
	}
	return o.APIKey
}

func (o *MoneycorpConfig) GetEndpoint() string {
	if o == nil {
		return ""
	}
	return o.Endpoint
}

func (o *MoneycorpConfig) GetPollingPeriod() *string {
	if o == nil {
		return nil
	}
	return o.PollingPeriod
}
