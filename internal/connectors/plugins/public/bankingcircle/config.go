package bankingcircle

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
)

type Config struct {
	Username              string `json:"username" yaml:"username" `
	Password              string `json:"password" yaml:"password" `
	Endpoint              string `json:"endpoint" yaml:"endpoint"`
	AuthorizationEndpoint string `json:"authorizationEndpoint" yaml:"authorizationEndpoint" `
	UserCertificate       string `json:"userCertificate" yaml:"userCertificate" `
	UserCertificateKey    string `json:"userCertificateKey" yaml:"userCertificateKey"`
}

func (c Config) validate() error {
	if c.Username == "" {
		return fmt.Errorf("missing username in config: %w", models.ErrInvalidConfig)
	}

	if c.Password == "" {
		return fmt.Errorf("missing password in config: %w", models.ErrInvalidConfig)
	}

	if c.Endpoint == "" {
		return fmt.Errorf("missing endpoint in config: %w", models.ErrInvalidConfig)
	}

	if c.AuthorizationEndpoint == "" {
		return fmt.Errorf("missing authorization endpoint in config: %w", models.ErrInvalidConfig)
	}

	if c.UserCertificate == "" {
		return fmt.Errorf("missing user certificate in config: %w", models.ErrInvalidConfig)
	}

	if c.UserCertificateKey == "" {
		return fmt.Errorf("missing user certificate key in config: %w", models.ErrInvalidConfig)
	}

	return nil
}

func unmarshalAndValidateConfig(payload []byte) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	return config, config.validate()
}
