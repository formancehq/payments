package stripe

import (
	"errors"
	"fmt"
	"time"
)

type Config struct {
	Pool          uint64        `json:"pool" yaml:"pool" bson:"pool"`
	PollingPeriod time.Duration `json:"pollingPeriod" yaml:"pollingPeriod" bson:"pollingPeriod"`
	PageSize      uint64        `json:"pageSize" yaml:"pageSize" bson:"pageSize"`
	ApiKey        string        `json:"apiKey" yaml:"apiKey" bson:"apiKey"`
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

type State struct {
	OldestID       string    `bson:"OldestID" json:"oldestID"`
	OldestDate     time.Time `bson:"oldestDate" json:"oldestDate"`
	MoreRecentID   string    `bson:"MoreRecentID" json:"moreRecentID"`
	MoreRecentDate time.Time `bson:"moreRecentDate" json:"moreRecentDate"`
}
