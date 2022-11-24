package wise

import "github.com/formancehq/payments/internal/pkg/configtemplate"

type Config struct {
	APIKey string `json:"apiKey" yaml:"apiKey" bson:"apiKey"`
}

func (c Config) Validate() error {
	if c.APIKey == "" {
		return ErrMissingAPIKey
	}

	return nil
}

func (c Config) BuildTemplate() (string, configtemplate.Config) {
	cfg := configtemplate.NewConfig()

	cfg.AddParameter("apiKey", configtemplate.TypeString, true)

	return connectorName, cfg
}
