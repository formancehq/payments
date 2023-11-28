package bankingcircle

import (
	"time"

	"github.com/formancehq/payments/cmd/connectors/internal/connectors"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/stack/libs/go-libs/logging"
)

type Loader struct{}

const allowedTasks = 50

func (l *Loader) AllowTasks() int {
	return allowedTasks
}

func (l *Loader) Name() models.ConnectorProvider {
	return Name
}

func (l *Loader) Load(logger logging.Logger, config Config) connectors.Connector {
	return newConnector(logger, config)
}

func (l *Loader) ApplyDefaults(cfg Config) Config {
	if cfg.PollingPeriod.Duration == 0 {
		cfg.PollingPeriod.Duration = 2 * time.Minute
	}

	if cfg.Name == "" {
		cfg.Name = Name.String()
	}

	return cfg
}

// NewLoader creates a new loader.
func NewLoader() *Loader {
	return &Loader{}
}
