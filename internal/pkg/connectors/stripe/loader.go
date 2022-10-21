package stripe

import (
	"time"

	"github.com/numary/payments/internal/pkg/integration"

	"github.com/numary/go-libs/sharedlogging"
)

type Loader struct{}

const allowedTasks = 50

func (l *Loader) AllowTasks() int {
	return allowedTasks
}

func (l *Loader) Name() string {
	return connectorName
}

func (l *Loader) Load(logger sharedlogging.Logger, config Config) integration.Connector[TaskDescriptor] {
	return newConnector(logger, config)
}

func (l *Loader) ApplyDefaults(cfg Config) Config {
	if cfg.PageSize == 0 {
		cfg.PageSize = 10
	}

	if cfg.PollingPeriod == 0 {
		cfg.PollingPeriod = 2 * time.Minute
	}

	return cfg
}

func NewLoader() *Loader {
	return &Loader{}
}
