package bankingcircle

import (
	"encoding/json"

	"github.com/formancehq/payments/pkg/connector"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	Username              string                     `json:"username" yaml:"username" validate:"required"`
	Password              string                     `json:"password" yaml:"password" validate:"required"`
	Endpoint              string                     `json:"endpoint" yaml:"endpoint" validate:"required"`
	AuthorizationEndpoint string                     `json:"authorizationEndpoint" yaml:"authorizationEndpoint" validate:"required"`
	UserCertificate       string                     `json:"userCertificate" yaml:"userCertificate" validate:"required"`
	UserCertificateKey    string                     `json:"userCertificateKey" yaml:"userCertificateKey" validate:"required"`
	PollingPeriod         connector.PollingPeriod `json:"pollingPeriod"`
}

const PAGE_SIZE = 100 // max page size is 5000 according to docs (!)

func unmarshalAndValidateConfig(payload []byte) (Config, error) {
	var raw struct {
		Username              string `json:"username"`
		Password              string `json:"password"`
		Endpoint              string `json:"endpoint"`
		AuthorizationEndpoint string `json:"authorizationEndpoint"`
		UserCertificate       string `json:"userCertificate"`
		UserCertificateKey    string `json:"userCertificateKey"`
		PollingPeriod         string `json:"pollingPeriod"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return Config{}, errors.Wrap(connector.ErrInvalidConfig, err.Error())
	}

	pp, err := connector.NewPollingPeriod(
		raw.PollingPeriod,
		connector.DefaultPollingPeriod,
		connector.MinimumPollingPeriod,
	)
	if err != nil {
		return Config{}, errors.Wrap(connector.ErrInvalidConfig, err.Error())
	}

	config := Config{
		Username:              raw.Username,
		Password:              raw.Password,
		Endpoint:              raw.Endpoint,
		AuthorizationEndpoint: raw.AuthorizationEndpoint,
		UserCertificate:       raw.UserCertificate,
		UserCertificateKey:    raw.UserCertificateKey,
		PollingPeriod:         pp,
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
