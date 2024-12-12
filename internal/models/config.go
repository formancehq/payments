package models

import (
	"encoding/json"
	"errors"
	"time"
)

const (
	defaultPollingPeriod = 2 * time.Minute
	defaultPageSize      = 25
)

type Config struct {
	Name          string        `json:"name"`
	PollingPeriod time.Duration `json:"pollingPeriod"`
	PageSize      int           `json:"pageSize"`
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
	if c.Name == "" {
		return errors.New("name is required")
	}

	if c.PollingPeriod.Seconds() < 30 {
		return errors.New("polling period must be at least 30 seconds")
	}

	return nil
}

func DefaultConfig() Config {
	return Config{
		PollingPeriod: defaultPollingPeriod,
		PageSize:      defaultPageSize,
	}
}
