package stripe

import (
	"time"

	"github.com/numary/payments/internal/pkg/integration"

	"github.com/numary/go-libs/sharedlogging"
)

type loader struct{}

const allowedTasks = 50

func (l *loader) AllowTasks() int {
	return allowedTasks
}

func (l *loader) Name() string {
	return connectorName
}

func (l *loader) Load(logger sharedlogging.Logger, config Config) integration.Connector[TaskDescriptor] {
	return newConnector(logger, config)
}

func (l *loader) ApplyDefaults(cfg Config) Config {
	if cfg.PageSize == 0 {
		cfg.PageSize = 10
	}

	if cfg.PollingPeriod == 0 {
		cfg.PollingPeriod = 2 * time.Minute
	}

	return cfg
}

func NewLoader() integration.Loader[Config, TaskDescriptor] {
	return &loader{}
}
