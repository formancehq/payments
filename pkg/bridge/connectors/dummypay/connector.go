package dummypay

import (
	"context"
	"fmt"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge/task"
)

// connectorName is the name of the connector.
const connectorName = "dummypay"

// Connector is the connector for the dummy payment connector.
type Connector struct {
	logger sharedlogging.Logger
	cfg    Config
	fs     fs
}

// Install executes post-installation steps to read and generate files.
// It is called after the connector is installed.
func (c *Connector) Install(ctx task.ConnectorContext[TaskDescriptor]) error {
	if err := ctx.Scheduler().Schedule(newTaskReadFiles(), true); err != nil {
		return fmt.Errorf("failed to schedule task to read files: %w", err)
	}

	if err := ctx.Scheduler().Schedule(newTaskGenerateFiles(), true); err != nil {
		return fmt.Errorf("failed to schedule task to generate files: %w", err)
	}

	return nil
}

// Uninstall executes pre-uninstallation steps to remove the generated files.
// It is called before the connector is uninstalled.
func (c *Connector) Uninstall(ctx context.Context) error {
	c.logger.Infof("Removing generated files from '%s'...", c.cfg.Directory)

	err := removeFiles(c.cfg, c.fs)
	if err != nil {
		return fmt.Errorf("failed to remove generated files: %w", err)
	}

	return nil
}

// Resolve resolves a task execution request based on the task descriptor.
func (c *Connector) Resolve(descriptor TaskDescriptor) task.Task {
	c.logger.Infof("Executing '%s' task...", descriptor.Key)

	return handleResolve(c.cfg, descriptor, c.fs)
}

// NewConnector creates a new dummy payment connector.
func NewConnector(logger sharedlogging.Logger, cfg Config, fs fs) *Connector {
	return &Connector{
		logger: logger.WithFields(map[string]any{
			"component": "connector",
		}),
		cfg: cfg,
		fs:  fs,
	}
}
