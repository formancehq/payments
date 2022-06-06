package stripe

import (
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge/integration"
	"net/http"
	"time"
)

type loader struct{}

func (l *loader) AllowTasks() int {
	return 5
}

func (l *loader) Name() string {
	return connectorName
}

func (l *loader) Load(logger sharedlogging.Logger, config Config) integration.Connector[TaskDescriptor, TimelineState] {
	client := NewDefaultClient(http.DefaultClient, config.ApiKey)
	return NewConnector(logger, client, config)
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

var _ integration.Loader[Config, TaskDescriptor, TimelineState] = &loader{}

func NewLoader() *loader {
	return &loader{}
}
