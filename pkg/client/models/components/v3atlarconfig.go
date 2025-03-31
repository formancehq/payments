// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

import (
	"github.com/formancehq/payments/pkg/client/internal/utils"
)

type V3AtlarConfig struct {
	AccessKey     string  `json:"accessKey"`
	BaseURL       string  `json:"baseUrl"`
	Name          string  `json:"name"`
	PageSize      *int64  `default:"25" json:"pageSize"`
	PollingPeriod *string `default:"2m" json:"pollingPeriod"`
	Provider      *string `default:"Atlar" json:"provider"`
	Secret        string  `json:"secret"`
}

func (v V3AtlarConfig) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(v, "", false)
}

func (v *V3AtlarConfig) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &v, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *V3AtlarConfig) GetAccessKey() string {
	if o == nil {
		return ""
	}
	return o.AccessKey
}

func (o *V3AtlarConfig) GetBaseURL() string {
	if o == nil {
		return ""
	}
	return o.BaseURL
}

func (o *V3AtlarConfig) GetName() string {
	if o == nil {
		return ""
	}
	return o.Name
}

func (o *V3AtlarConfig) GetPageSize() *int64 {
	if o == nil {
		return nil
	}
	return o.PageSize
}

func (o *V3AtlarConfig) GetPollingPeriod() *string {
	if o == nil {
		return nil
	}
	return o.PollingPeriod
}

func (o *V3AtlarConfig) GetProvider() *string {
	if o == nil {
		return nil
	}
	return o.Provider
}

func (o *V3AtlarConfig) GetSecret() string {
	if o == nil {
		return ""
	}
	return o.Secret
}
