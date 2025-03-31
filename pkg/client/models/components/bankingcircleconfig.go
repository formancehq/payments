// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

import (
	"github.com/formancehq/payments/pkg/client/internal/utils"
)

type BankingCircleConfig struct {
	Name                  string  `json:"name"`
	Provider              *string `default:"Bankingcircle" json:"provider"`
	Username              string  `json:"username"`
	Password              string  `json:"password"`
	Endpoint              string  `json:"endpoint"`
	AuthorizationEndpoint string  `json:"authorizationEndpoint"`
	UserCertificate       string  `json:"userCertificate"`
	UserCertificateKey    string  `json:"userCertificateKey"`
	// The frequency at which the connector will try to fetch new BalanceTransaction objects from Banking Circle API.
	//
	PollingPeriod *string `default:"120s" json:"pollingPeriod"`
}

func (b BankingCircleConfig) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(b, "", false)
}

func (b *BankingCircleConfig) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &b, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *BankingCircleConfig) GetName() string {
	if o == nil {
		return ""
	}
	return o.Name
}

func (o *BankingCircleConfig) GetProvider() *string {
	if o == nil {
		return nil
	}
	return o.Provider
}

func (o *BankingCircleConfig) GetUsername() string {
	if o == nil {
		return ""
	}
	return o.Username
}

func (o *BankingCircleConfig) GetPassword() string {
	if o == nil {
		return ""
	}
	return o.Password
}

func (o *BankingCircleConfig) GetEndpoint() string {
	if o == nil {
		return ""
	}
	return o.Endpoint
}

func (o *BankingCircleConfig) GetAuthorizationEndpoint() string {
	if o == nil {
		return ""
	}
	return o.AuthorizationEndpoint
}

func (o *BankingCircleConfig) GetUserCertificate() string {
	if o == nil {
		return ""
	}
	return o.UserCertificate
}

func (o *BankingCircleConfig) GetUserCertificateKey() string {
	if o == nil {
		return ""
	}
	return o.UserCertificateKey
}

func (o *BankingCircleConfig) GetPollingPeriod() *string {
	if o == nil {
		return nil
	}
	return o.PollingPeriod
}
