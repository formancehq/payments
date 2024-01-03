package dummypay

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/cmd/connectors/internal/connectors"
	"github.com/formancehq/payments/cmd/connectors/internal/task"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/stack/libs/go-libs/logging"
)

// Name is the name of the connector.
const Name = models.ConnectorProviderDummyPay

// Connector is the connector for the dummy payment connector.
type Connector struct {
	logger logging.Logger
	cfg    Config
	fs     fs
}

func newConnector(logger logging.Logger, cfg Config, fs fs) *Connector {
	return &Connector{
		logger: logger.WithFields(map[string]any{
			"component": "connector",
		}),
		cfg: cfg,
		fs:  fs,
	}
}

func (c *Connector) UpdateConfig(ctx task.ConnectorContext, config models.ConnectorConfigObject) error {
	cfg, ok := config.(Config)
	if !ok {
		return connectors.ErrInvalidConfig
	}

	c.cfg = cfg

	return nil
}

// Install executes post-installation steps to read and generate files.
// It is called after the connector is installed.
func (c *Connector) Install(ctx task.ConnectorContext) error {
	initDirectoryDescriptor, err := models.EncodeTaskDescriptor(newTaskGenerateFiles())
	if err != nil {
		return fmt.Errorf("failed to create generate files task descriptor: %w", err)
	}

	if err = ctx.Scheduler().Schedule(ctx.Context(), initDirectoryDescriptor, models.TaskSchedulerOptions{
		ScheduleOption: models.OPTIONS_RUN_NOW_SYNC,
		RestartOption:  models.OPTIONS_RESTART_NEVER,
	}); err != nil {
		return fmt.Errorf("failed to schedule task to generate files: %w", err)
	}

	readFilesDescriptor, err := models.EncodeTaskDescriptor(newTaskReadFiles())
	if err != nil {
		return fmt.Errorf("failed to create read files task descriptor: %w", err)
	}

	if err = ctx.Scheduler().Schedule(ctx.Context(), readFilesDescriptor, models.TaskSchedulerOptions{
		ScheduleOption: models.OPTIONS_RUN_PERIODICALLY,
		Duration:       c.cfg.FilePollingPeriod.Duration,
		// No need to restart this task, since the connector is not existing or
		// was uninstalled previously, the task does not exists in the database
		RestartOption: models.OPTIONS_RESTART_NEVER,
	}); err != nil {
		return fmt.Errorf("failed to schedule task to read files: %w", err)
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
func (c *Connector) Resolve(descriptor models.TaskDescriptor) task.Task {
	taskDescriptor, err := models.DecodeTaskDescriptor[TaskDescriptor](descriptor)
	if err != nil {
		panic(err)
	}

	c.logger.Infof("Executing '%s' task...", taskDescriptor.Key)

	return handleResolve(c.cfg, taskDescriptor, c.fs)
}

func (c *Connector) SupportedCurrenciesAndDecimals() map[string]int {
	return supportedCurrenciesWithDecimal
}

func (c *Connector) InitiatePayment(ctx task.ConnectorContext, transfer *models.TransferInitiation) error {
	// TODO implement me
	return connectors.ErrNotImplemented
}

func (c *Connector) CreateExternalBankAccount(ctx task.ConnectorContext, bankAccount *models.BankAccount) error {
	// TODO implement me
	return connectors.ErrNotImplemented
}

var _ connectors.Connector = &Connector{}
