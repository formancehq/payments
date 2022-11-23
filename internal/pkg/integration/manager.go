package integration

import (
	"context"

	"github.com/formancehq/go-libs/sharedlogging"
	"github.com/formancehq/payments/internal/pkg/payments"
	"github.com/formancehq/payments/internal/pkg/task"
	"github.com/pkg/errors"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrAlreadyInstalled = errors.New("already installed")
	ErrNotInstalled     = errors.New("not installed")
	ErrNotEnabled       = errors.New("not enabled")
	ErrAlreadyRunning   = errors.New("already running")
)

type ConnectorManager[
	Config payments.ConnectorConfigObject,
	TaskDescriptor payments.TaskDescriptor,
] struct {
	logger           sharedlogging.Logger
	loader           Loader[Config, TaskDescriptor]
	connector        Connector[TaskDescriptor]
	store            ConnectorStore
	schedulerFactory TaskSchedulerFactory[TaskDescriptor]
	scheduler        *task.DefaultTaskScheduler[TaskDescriptor]
}

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor]) Enable(ctx context.Context) error {
	l.logger.Info("Enabling connector")

	err := l.store.Enable(ctx, l.loader.Name())
	if err != nil {
		return err
	}

	return nil
}

func (l *ConnectorManager[ConnectorConfig,
	TaskDescriptor]) ReadConfig(ctx context.Context,
) (*ConnectorConfig, error) {
	var config ConnectorConfig

	err := l.store.ReadConfig(ctx, l.loader.Name(), &config)
	if err != nil {
		return &config, err
	}

	config = l.loader.ApplyDefaults(config)

	return &config, nil
}

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor]) load(config ConnectorConfig) {
	l.connector = l.loader.Load(l.logger, config)
	l.scheduler = l.schedulerFactory.Make(l.connector, l.loader.AllowTasks())
}

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor]) Install(ctx context.Context, config ConnectorConfig) error {
	l.logger.WithFields(map[string]interface{}{
		"config": config,
	}).Infof("Install connector %s", l.loader.Name())

	isInstalled, err := l.store.IsInstalled(ctx, l.loader.Name())
	if err != nil {
		l.logger.Errorf("Error checking if connector is installed: %s", err)

		return err
	}

	if isInstalled {
		l.logger.Errorf("Connector already installed")

		return ErrAlreadyInstalled
	}

	config = l.loader.ApplyDefaults(config)

	if err = config.Validate(); err != nil {
		return err
	}

	l.load(config)

	err = l.connector.Install(task.NewConnectorContext[TaskDescriptor](context.Background(), l.scheduler))
	if err != nil {
		l.logger.Errorf("Error starting connector: %s", err)

		return err
	}

	err = l.store.Install(ctx, l.loader.Name(), config)
	if err != nil {
		return err
	}

	l.logger.Infof("Connector installed")

	return nil
}

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor]) Uninstall(ctx context.Context) error {
	l.logger.Infof("Uninstalling connector")

	isInstalled, err := l.IsInstalled(ctx)
	if err != nil {
		l.logger.Errorf("Error checking if connector is installed: %s", err)

		return err
	}

	if !isInstalled {
		l.logger.Errorf("Connector not installed")

		return ErrNotInstalled
	}

	err = l.scheduler.Shutdown(ctx)
	if err != nil {
		return err
	}

	err = l.connector.Uninstall(ctx)
	if err != nil {
		return err
	}

	err = l.store.Uninstall(ctx, l.loader.Name())
	if err != nil {
		return err
	}

	l.logger.Info("Connector uninstalled")

	return nil
}

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor]) Restore(ctx context.Context) error {
	l.logger.Info("Restoring state")

	installed, err := l.IsInstalled(ctx)
	if err != nil {
		return err
	}

	if !installed {
		l.logger.Info("Not installed, skip")

		return ErrNotInstalled
	}

	enabled, err := l.IsEnabled(ctx)
	if err != nil {
		return err
	}

	if !enabled {
		l.logger.Info("Not enabled, skip")

		return ErrNotEnabled
	}

	if l.connector != nil {
		return ErrAlreadyRunning
	}

	config, err := l.ReadConfig(ctx)
	if err != nil {
		return err
	}

	l.load(*config)

	err = l.scheduler.Restore(ctx)
	if err != nil {
		l.logger.Errorf("Unable to restore scheduler: %s", err)

		return err
	}

	l.logger.Info("State restored")

	return nil
}

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor]) Disable(ctx context.Context) error {
	l.logger.Info("Disabling connector")

	return l.store.Disable(ctx, l.loader.Name())
}

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor]) IsEnabled(ctx context.Context) (bool, error) {
	return l.store.IsEnabled(ctx, l.loader.Name())
}

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor]) FindAll(ctx context.Context) ([]payments.ConnectorBaseInfo, error) {
	return l.store.FindAll(ctx)
}

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor]) IsInstalled(ctx context.Context) (bool, error) {
	return l.store.IsInstalled(ctx, l.loader.Name())
}

func (l *ConnectorManager[ConnectorConfig,
	TaskDescriptor]) ListTasksStates(ctx context.Context,
) ([]payments.TaskState[TaskDescriptor], error) {
	return l.scheduler.ListTasks(ctx)
}

func (l *ConnectorManager[Config, TaskDescriptor]) ReadTaskState(ctx context.Context,
	descriptor TaskDescriptor,
) (*payments.TaskState[TaskDescriptor], error) {
	return l.scheduler.ReadTask(ctx, descriptor)
}

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor]) Reset(ctx context.Context) error {
	config, err := l.ReadConfig(ctx)
	if err != nil {
		return err
	}

	err = l.Uninstall(ctx)
	if err != nil {
		return err
	}

	return l.Install(ctx, *config)
}

func NewConnectorManager[
	ConnectorConfig payments.ConnectorConfigObject,
	TaskDescriptor payments.TaskDescriptor,
](
	logger sharedlogging.Logger,
	store ConnectorStore,
	loader Loader[ConnectorConfig, TaskDescriptor],
	schedulerFactory TaskSchedulerFactory[TaskDescriptor],
) *ConnectorManager[ConnectorConfig, TaskDescriptor] {
	return &ConnectorManager[ConnectorConfig, TaskDescriptor]{
		logger: logger.WithFields(map[string]interface{}{
			"component": "connector-manager",
			"provider":  loader.Name(),
		}),
		store:            store,
		loader:           loader,
		schedulerFactory: schedulerFactory,
	}
}
