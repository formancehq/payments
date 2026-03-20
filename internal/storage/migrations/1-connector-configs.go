package migrations

import (
	"encoding/json"
)

const (
	defaultPageSize = 25
)

func transformV2ConfigToV3Config(provider string, v2Config json.RawMessage) (bool, json.RawMessage, error) {
	switch provider {
	case "ADYEN":
		v3Config, err := transformV2AdyenConfigToV3(v2Config)
		return true, v3Config, err
	case "ATLAR":
		v3Config, err := transformV2AtlarConfigToV3(v2Config)
		return true, v3Config, err
	case "BANKING-CIRCLE":
		v3Config, err := transformV2BankingCircleConfigToV3(v2Config)
		return true, v3Config, err
	case "CURRENCY-CLOUD":
		v3Config, err := transformV2CurrencyCloudConfigToV3(v2Config)
		return true, v3Config, err
	case "GENERIC":
		v3Config, err := transformV2GenericConfigToV3(v2Config)
		return true, v3Config, err
	case "MANGOPAY":
		v3Config, err := transformV2MangopayConfigToV3(v2Config)
		return true, v3Config, err
	case "MODULR":
		v3Config, err := transformV2ModulrConfigToV3(v2Config)
		return true, v3Config, err
	case "MONEYCORP":
		v3Config, err := transformV2MoneycorpConfigToV3(v2Config)
		return true, v3Config, err
	case "STRIPE":
		v3Config, err := transformV2StripeConfigToV3(v2Config)
		return true, v3Config, err
	case "WISE":
		v3Config, err := transformV2WiseConfigToV3(v2Config)
		return true, v3Config, err
	default:
		return false, nil, nil
	}
}

func transformV2AdyenConfigToV3(v2Config json.RawMessage) (json.RawMessage, error) {
	var v2 v2AdyenConfig
	if err := json.Unmarshal(v2Config, &v2); err != nil {
		return nil, err
	}

	v3Config := v3AdyenConfig{
		APIKey:             v2.APIKey,
		LiveEndpointPrefix: v2.LiveEndpointPrefix,
	}

	type res struct {
		v3DefaultConfig
		v3AdyenConfig
	}

	return json.Marshal(res{
		v3DefaultConfig: v3DefaultConfig{
			Name:          v2.Name,
			PollingPeriod: v2.PollingPeriod.String(),
			PageSize:      defaultPageSize,
		},
		v3AdyenConfig: v3Config,
	})
}

func transformV2AtlarConfigToV3(v2Config json.RawMessage) (json.RawMessage, error) {
	var v2 v2AtlarConfig
	if err := json.Unmarshal(v2Config, &v2); err != nil {
		return nil, err
	}

	v3Config := v3AtlarConfig{
		BaseURL:   v2.BaseUrl,
		AccessKey: v2.AccessKey,
		Secret:    v2.Secret,
	}

	type res struct {
		v3DefaultConfig
		v3AtlarConfig
	}

	return json.Marshal(res{
		v3DefaultConfig: v3DefaultConfig{
			Name:          v2.Name,
			PollingPeriod: v2.PollingPeriod.String(),
			PageSize:      int(v2.PageSize),
		},
		v3AtlarConfig: v3Config,
	})
}

func transformV2BankingCircleConfigToV3(v2Config json.RawMessage) (json.RawMessage, error) {
	var v2 v2BankingCircleConfig
	if err := json.Unmarshal(v2Config, &v2); err != nil {
		return nil, err
	}

	v3Config := v3BankingCircleConfig{
		Username:              v2.Username,
		Password:              v2.Password,
		Endpoint:              v2.Endpoint,
		AuthorizationEndpoint: v2.AuthorizationEndpoint,
		UserCertificate:       v2.UserCertificate,
		UserCertificateKey:    v2.UserCertificateKey,
	}

	type res struct {
		v3DefaultConfig
		v3BankingCircleConfig
	}

	return json.Marshal(res{
		v3DefaultConfig: v3DefaultConfig{
			Name:          v2.Name,
			PollingPeriod: v2.PollingPeriod.String(),
			PageSize:      defaultPageSize,
		},
		v3BankingCircleConfig: v3Config,
	})
}

func transformV2CurrencyCloudConfigToV3(v2Config json.RawMessage) (json.RawMessage, error) {
	var v2 v2CurrencyCloudConfig
	if err := json.Unmarshal(v2Config, &v2); err != nil {
		return nil, err
	}

	v3Config := v3CurrencyCloudConfig{
		LoginID:  v2.LoginID,
		APIKey:   v2.APIKey,
		Endpoint: v2.Endpoint,
	}

	type res struct {
		v3DefaultConfig
		v3CurrencyCloudConfig
	}

	return json.Marshal(res{
		v3DefaultConfig: v3DefaultConfig{
			Name:          v2.Name,
			PollingPeriod: v2.PollingPeriod.String(),
			PageSize:      defaultPageSize,
		},
		v3CurrencyCloudConfig: v3Config,
	})
}

func transformV2GenericConfigToV3(v2Config json.RawMessage) (json.RawMessage, error) {
	var v2 v2GenericConfig
	if err := json.Unmarshal(v2Config, &v2); err != nil {
		return nil, err
	}

	v3Config := v3GenericConfig{
		APIKey:   v2.APIKey,
		Endpoint: v2.Endpoint,
	}

	type res struct {
		v3DefaultConfig
		v3GenericConfig
	}

	return json.Marshal(res{
		v3DefaultConfig: v3DefaultConfig{
			Name:          v2.Name,
			PollingPeriod: v2.PollingPeriod.String(),
			PageSize:      defaultPageSize,
		},
		v3GenericConfig: v3Config,
	})
}

func transformV2MangopayConfigToV3(v2Config json.RawMessage) (json.RawMessage, error) {
	var v2 v2MangopayConfig
	if err := json.Unmarshal(v2Config, &v2); err != nil {
		return nil, err
	}

	v3Config := v3MangopayConfig{
		ClientID: v2.ClientID,
		APIKey:   v2.APIKey,
		Endpoint: v2.Endpoint,
	}

	type res struct {
		v3DefaultConfig
		v3MangopayConfig
	}

	return json.Marshal(res{
		v3DefaultConfig: v3DefaultConfig{
			Name:          v2.Name,
			PollingPeriod: v2.PollingPeriod.String(),
			PageSize:      defaultPageSize,
		},
		v3MangopayConfig: v3Config,
	})
}

func transformV2ModulrConfigToV3(v2Config json.RawMessage) (json.RawMessage, error) {
	var v2 v2ModulrConfig
	if err := json.Unmarshal(v2Config, &v2); err != nil {
		return nil, err
	}

	v3Config := v3ModulrConfig{
		APIKey:    v2.APIKey,
		APISecret: v2.APISecret,
		Endpoint:  v2.Endpoint,
	}

	type res struct {
		v3DefaultConfig
		v3ModulrConfig
	}

	return json.Marshal(res{
		v3DefaultConfig: v3DefaultConfig{
			Name:          v2.Name,
			PollingPeriod: v2.PollingPeriod.String(),
			PageSize:      v2.PageSize,
		},
		v3ModulrConfig: v3Config,
	})
}

func transformV2MoneycorpConfigToV3(v2Config json.RawMessage) (json.RawMessage, error) {
	var v2 v2MoneycorpConfig
	if err := json.Unmarshal(v2Config, &v2); err != nil {
		return nil, err
	}

	v3Config := v3MoneycorpConfig{
		ClientID: v2.ClientID,
		APIKey:   v2.APIKey,
		Endpoint: v2.Endpoint,
	}

	type res struct {
		v3DefaultConfig
		v3MoneycorpConfig
	}

	return json.Marshal(res{
		v3DefaultConfig: v3DefaultConfig{
			Name:          v2.Name,
			PollingPeriod: v2.PollingPeriod.String(),
			PageSize:      defaultPageSize,
		},
		v3MoneycorpConfig: v3Config,
	})
}

func transformV2StripeConfigToV3(v2Config json.RawMessage) (json.RawMessage, error) {
	var v2 v2StripeConfig
	if err := json.Unmarshal(v2Config, &v2); err != nil {
		return nil, err
	}

	v3Config := v3StripeConfig{
		APIKey: v2.APIKey,
	}

	type res struct {
		v3DefaultConfig
		v3StripeConfig
	}

	return json.Marshal(res{
		v3DefaultConfig: v3DefaultConfig{
			Name:          v2.Name,
			PollingPeriod: v2.PollingPeriod.String(),
			PageSize:      int(v2.PageSize),
		},
		v3StripeConfig: v3Config,
	})
}

func transformV2WiseConfigToV3(v2Config json.RawMessage) (json.RawMessage, error) {
	var v2 v2WiseConfig
	if err := json.Unmarshal(v2Config, &v2); err != nil {
		return nil, err
	}

	v3Config := v3WiseConfig{
		APIKey: v2.APIKey,
	}

	type res struct {
		v3DefaultConfig
		v3WiseConfig
	}

	return json.Marshal(res{
		v3DefaultConfig: v3DefaultConfig{
			Name:          v2.Name,
			PollingPeriod: v2.PollingPeriod.String(),
			PageSize:      defaultPageSize,
		},
		v3WiseConfig: v3Config,
	})
}

type v2AdyenConfig struct {
	Name               string     `json:"name"`
	APIKey             string     `json:"apiKey"`
	HMACKey            string     `json:"hmacKey"`
	LiveEndpointPrefix string     `json:"liveEndpointPrefix"`
	PollingPeriod      v2Duration `json:"pollingPeriod"`
}

type v2AtlarConfig struct {
	Name                                  string     `json:"name"`
	PollingPeriod                         v2Duration `json:"pollingPeriod"`
	TransferInitiationStatusPollingPeriod v2Duration `json:"transferInitiationStatusPollingPeriod"`
	BaseUrl                               string     `json:"baseURL"`
	AccessKey                             string     `json:"accessKey"`
	Secret                                string     `json:"secret"`
	PageSize                              uint64     `json:"pageSize"`
}

type v2BankingCircleConfig struct {
	Name                  string     `json:"name"`
	Username              string     `json:"username"`
	Password              string     `json:"password"`
	Endpoint              string     `json:"endpoint"`
	AuthorizationEndpoint string     `json:"authorizationEndpoint"`
	UserCertificate       string     `json:"userCertificate"`
	UserCertificateKey    string     `json:"userCertificateKey"`
	PollingPeriod         v2Duration `json:"pollingPeriod"`
}

type v2CurrencyCloudConfig struct {
	Name          string     `json:"name"`
	LoginID       string     `json:"loginID"`
	APIKey        string     `json:"apiKey"`
	Endpoint      string     `json:"endpoint"`
	PollingPeriod v2Duration `json:"pollingPeriod"`
}

type v2GenericConfig struct {
	Name          string     `json:"name"`
	APIKey        string     `json:"apiKey"`
	Endpoint      string     `json:"endpoint"`
	PollingPeriod v2Duration `json:"pollingPeriod"`
}

type v2MangopayConfig struct {
	Name          string     `json:"name"`
	ClientID      string     `json:"clientID"`
	APIKey        string     `json:"apiKey"`
	Endpoint      string     `json:"endpoint"`
	PollingPeriod v2Duration `json:"pollingPeriod"`
}

type v2ModulrConfig struct {
	Name          string     `json:"name"`
	APIKey        string     `json:"apiKey"`
	APISecret     string     `json:"apiSecret"`
	Endpoint      string     `json:"endpoint"`
	PollingPeriod v2Duration `json:"pollingPeriod"`
	PageSize      int        `json:"pageSize"`
}

type v2MoneycorpConfig struct {
	Name          string     `json:"name"`
	ClientID      string     `json:"clientID"`
	APIKey        string     `json:"apiKey"`
	Endpoint      string     `json:"endpoint"`
	PollingPeriod v2Duration `json:"pollingPeriod"`
}

type v2StripeConfig struct {
	Name          string     `json:"name"`
	PollingPeriod v2Duration `json:"pollingPeriod"`
	APIKey        string     `json:"apiKey"`
	PageSize      uint64     `json:"pageSize"`
}

type v2WiseConfig struct {
	Name          string     `json:"name"`
	APIKey        string     `json:"apiKey"`
	PollingPeriod v2Duration `json:"pollingPeriod"`
}

type v3DefaultConfig struct {
	Name          string `json:"name"`
	PollingPeriod string `json:"pollingPeriod"`
	PageSize      int    `json:"pageSize"`
}

type v3AdyenConfig struct {
	APIKey             string `json:"apiKey"`
	WebhookUsername    string `json:"webhookUsername"`
	WebhookPassword    string `json:"webhookPassword"`
	CompanyID          string `json:"companyID"`
	LiveEndpointPrefix string `json:"liveEndpointPrefix"`
}

type v3AtlarConfig struct {
	BaseURL   string `json:"baseURL"`
	AccessKey string `json:"accessKey"`
	Secret    string `json:"secret"`
}

type v3BankingCircleConfig struct {
	Username              string `json:"username" yaml:"username" `
	Password              string `json:"password" yaml:"password" `
	Endpoint              string `json:"endpoint" yaml:"endpoint"`
	AuthorizationEndpoint string `json:"authorizationEndpoint" yaml:"authorizationEndpoint" `
	UserCertificate       string `json:"userCertificate" yaml:"userCertificate" `
	UserCertificateKey    string `json:"userCertificateKey" yaml:"userCertificateKey"`
}

type v3CurrencyCloudConfig struct {
	LoginID  string `json:"loginID"`
	APIKey   string `json:"apiKey"`
	Endpoint string `json:"endpoint"`
}

type v3GenericConfig struct {
	APIKey   string `json:"apiKey"`
	Endpoint string `json:"endpoint"`
}

type v3MangopayConfig struct {
	ClientID string `json:"clientID"`
	APIKey   string `json:"apiKey"`
	Endpoint string `json:"endpoint"`
}

type v3ModulrConfig struct {
	APIKey    string `json:"apiKey"`
	APISecret string `json:"apiSecret"`
	Endpoint  string `json:"endpoint"`
}

type v3MoneycorpConfig struct {
	ClientID string `json:"clientID"`
	APIKey   string `json:"apiKey"`
	Endpoint string `json:"endpoint"`
}

type v3StripeConfig struct {
	APIKey string `json:"apiKey"`
}

type v3WiseConfig struct {
	APIKey           string `json:"apiKey"`
	WebhookPublicKey string `json:"webhookPublicKey"`
}
