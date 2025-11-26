package models

import (
	"encoding/json"
	"time"

	"github.com/go-playground/validator/v10"
)

const (
	defaultPollingPeriod = 30 * time.Minute
)

// since the json unmarshaller is case-insensitive this generic interface will be used to receive back the unmarshaled struct from a plugin
// then we can remarshal the payload and enforce the case we expect (eg. clientID vs clientId) rather than blindly trusting the raw user input
type PluginInternalConfig interface{}

// Config is the generic configuration that all connectors share
// Note that the PollingPeriod defined here is often overwritten by the connector-specific configuration
type Config struct {
	Name          string        `json:"name" validate:"required,gte=3,lte=500"`
	PollingPeriod time.Duration `json:"pollingPeriod" validate:"required,gte=1200000000000,lte=86400000000000"` // gte=20mn lte=1d in ns
}

func (c Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name          string `json:"name"`
		PollingPeriod string `json:"pollingPeriod"`
	}{
		Name:          c.Name,
		PollingPeriod: c.PollingPeriod.String(),
	})
}

func (c *Config) UnmarshalJSON(data []byte) error {
	var raw struct {
		Name          string `json:"name"`
		PollingPeriod string `json:"pollingPeriod"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	pollingPeriod := defaultPollingPeriod
	if raw.PollingPeriod != "" {
		p, err := time.ParseDuration(raw.PollingPeriod)
		if err != nil {
			return err
		}
		pollingPeriod = p
	}

	c.Name = raw.Name

	if pollingPeriod > 0 {
		c.PollingPeriod = pollingPeriod
	}

	return nil
}

func (c Config) Validate() error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	return validate.Struct(c)
}

func DefaultConfig() Config {
	return Config{
		PollingPeriod: defaultPollingPeriod,
	}
}
