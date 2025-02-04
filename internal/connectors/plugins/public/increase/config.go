package increase

import (
	"fmt"
	"time"
)

const (
	minPollingPeriod = 30 * time.Second
)

type Config struct {
	APIKey        string        `json:"apiKey" yaml:"apiKey"`
	PollingPeriod time.Duration `json:"pollingPeriod" yaml:"pollingPeriod"`
}

func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}
	if c.PollingPeriod < minPollingPeriod {
		return fmt.Errorf("pollingPeriod must be at least %v", minPollingPeriod)
	}
	return nil
}
