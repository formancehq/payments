package wise

import (
	"github.com/formancehq/payments/internal/pkg/integration"
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
	return cfg
}

// NewLoader creates a new loader.
func NewLoader() *Loader {
	return &Loader{}
}
