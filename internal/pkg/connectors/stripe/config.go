package stripe

import (
	"errors"
	"fmt"

	"github.com/formancehq/payments/internal/pkg/connectors"
)

type Config struct {
	PollingPeriod  connectors.Duration `json:"pollingPeriod" yaml:"pollingPeriod" bson:"pollingPeriod"`
	APIKey         string              `json:"apiKey" yaml:"apiKey" bson:"apiKey"`
	TimelineConfig `bson:",inline"`
}

func (c Config) String() string {
	return fmt.Sprintf("pollingPeriod=%d, pageSize=%d, apiKey=%s", c.PollingPeriod, c.PageSize, c.APIKey)
}

func (c Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("missing api key")
	}

	return nil
}

type TimelineConfig struct {
	PageSize uint64 `json:"pageSize" yaml:"pageSize" bson:"pageSize"`
}
