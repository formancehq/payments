package dummypay

import (
	"time"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge/integration"
)

type loader struct{}

// Name returns the name of the connector.
func (l *loader) Name() string {
	return connectorName
}

// AllowTasks returns the amount of tasks that are allowed to be scheduled.
func (l *loader) AllowTasks() int {
	return 10
}

const (
	// defaultFilePollingPeriod is the default period between file polling.
	defaultFilePollingPeriod = 10 * time.Second

	// defaultFileGenerationPeriod is the default period between file generation.
	defaultFileGenerationPeriod = 5 * time.Second
)

// ApplyDefaults applies default values to the configuration.
func (l *loader) ApplyDefaults(cfg Config) Config {
	if cfg.FileGenerationPeriod == 0 {
		cfg.FileGenerationPeriod = defaultFileGenerationPeriod
	}

	if cfg.FilePollingPeriod == 0 {
		cfg.FilePollingPeriod = defaultFilePollingPeriod
	}

	return cfg
}

// Load returns the connector.
func (l *loader) Load(logger sharedlogging.Logger, config Config) integration.Connector[TaskDescriptor] {
	return NewConnector(logger, config, newFS())
}

// NewLoader creates a new loader.
func NewLoader() integration.Loader[Config, TaskDescriptor] {
	return &loader{}
}
