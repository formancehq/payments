package stripe

import (
	"errors"
	"fmt"
	"time"
)

type Config struct {
	Pool           int           `json:"pool" yaml:"pool" bson:"pool"`
	PollingPeriod  time.Duration `json:"pollingPeriod" yaml:"pollingPeriod" bson:"pollingPeriod"`
	ApiKey         string        `json:"apiKey" yaml:"apiKey" bson:"apiKey"`
	TimelineConfig `bson:",inline"`
}

func (c *Config) String() string {
	return fmt.Sprintf("pool=%d, pollingPeriod=%d, pageSize=%d, apiKey=%s", c.Pool, c.PollingPeriod, c.PageSize, c.ApiKey)
}

func (c Config) Validate() error {
	if c.ApiKey == "" {
		return errors.New("missing api key")
	}
	return nil
}

type TimelineConfig struct {
	PageSize uint64 `json:"pageSize" yaml:"pageSize" bson:"pageSize"`
}
