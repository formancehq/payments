package bankingcircle

import (
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	Username              string `json:"username" yaml:"username" validate:"required"`
	Password              string `json:"password" yaml:"password" validate:"required"`
	Endpoint              string `json:"endpoint" yaml:"endpoint" validate:"required"`
	AuthorizationEndpoint string `json:"authorizationEndpoint" yaml:"authorizationEndpoint" validate:"required"`
	UserCertificate       string `json:"userCertificate" yaml:"userCertificate" validate:"required"`
	UserCertificateKey    string `json:"userCertificateKey" yaml:"userCertificateKey" validate:"required"`
}

func unmarshalAndValidateConfig(payload []byte) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
