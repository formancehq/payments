package models

import (
	"encoding/json"
	"time"

	"github.com/go-playground/validator/v10"
)

const (
	defaultPollingPeriod = 2 * time.Minute
	defaultPageSize      = 25
)

// since the json unmarshaller is case-insensitive this generic interface will be used to receive back the unmarshaled struct from a plugin
// then we can remarshal the payload and enforce the case we expect (eg. clientID vs clientId) rather than blindly trusting the raw user input
type PluginInternalConfig interface{}

// Config is the generic configuration that all connectors share
type Config struct {
	Name          string        `json:"name" validate:"required,gte=3,lte=500"`
	PollingPeriod time.Duration `json:"pollingPeriod" validate:"required,gte=30000000000,lte=86400000000000"` // gte=30s lte=1d in ns
	PageSize      int           `json:"pageSize" validate:"lte=150"`
}

func (c Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name          string `json:"name"`
		PollingPeriod string `json:"pollingPeriod"`
		PageSize      int    `json:"pageSize"`
	}{
		Name:          c.Name,
		PollingPeriod: c.PollingPeriod.String(),
		PageSize:      c.PageSize,
	})
}

func (c *Config) UnmarshalJSON(data []byte) error {
	var raw struct {
		Name          string `json:"name"`
		PollingPeriod string `json:"pollingPeriod"`
		PageSize      int    `json:"pageSize"`
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

	if raw.PageSize > 0 {
		c.PageSize = raw.PageSize
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
		PageSize:      defaultPageSize,
	}
}
