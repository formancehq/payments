package connectors

import (
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
)

type Configurer struct {
	connectorPollingPeriodDefault time.Duration
	connectorPollingPeriodMinimum time.Duration
}

func NewConfigurer(pollingPeriodDefault, pollingPeriodMinimum time.Duration) *Configurer {
	return &Configurer{
		connectorPollingPeriodDefault: pollingPeriodDefault,
		connectorPollingPeriodMinimum: pollingPeriodMinimum,
	}
}

func (c *Configurer) DefaultConfig() models.Config {
	return models.Config{
		PollingPeriod: c.connectorPollingPeriodDefault,
	}
}

func (c *Configurer) Validate(conf models.Config) error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(conf); err != nil {
		return fmt.Errorf("%w: %w", models.ErrInvalidConfig, err)
	}

	if conf.PollingPeriod < c.connectorPollingPeriodMinimum {
		return fmt.Errorf("%w: polling period cannot be lower than minimum of %s", ErrPollingPeriod, c.connectorPollingPeriodMinimum)
	}
	return nil
}
