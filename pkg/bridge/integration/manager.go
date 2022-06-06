package integration

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	payments "github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge/task"
	"github.com/pkg/errors"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrAlreadyInstalled = errors.New("already installed")
	ErrNotInstalled     = errors.New("not installed")
	ErrNotEnabled       = errors.New("not enabled")
	ErrAlreadyRunning   = errors.New("already running")
)

type TaskSchedulerFactory[TaskDescriptor payments.TaskDescriptor, TaskState any] interface {
	Make(resolver task.Resolver[TaskDescriptor, TaskState], maxTasks int) *task.DefaultTaskScheduler[TaskDescriptor, TaskState]
}
type TaskSchedulerFactoryFn[TaskDescriptor payments.TaskDescriptor, TaskState any] func(resolver task.Resolver[TaskDescriptor, TaskState], maxProcesses int) *task.DefaultTaskScheduler[TaskDescriptor, TaskState]

func (fn TaskSchedulerFactoryFn[TaskDescriptor, TaskState]) Make(resolver task.Resolver[TaskDescriptor, TaskState], maxTasks int) *task.DefaultTaskScheduler[TaskDescriptor, TaskState] {
	return fn(resolver, maxTasks)
}

type ConnectorManager[
	Config payments.ConnectorConfigObject,
	TaskDescriptor payments.TaskDescriptor,
	TaskState any,
] struct {
	logger           sharedlogging.Logger
	loader           Loader[Config, TaskDescriptor, TaskState]
	connector        Connector[TaskDescriptor, TaskState]
	store            ConnectorStore
	schedulerFactory TaskSchedulerFactory[TaskDescriptor, TaskState]
	scheduler        *task.DefaultTaskScheduler[TaskDescriptor, TaskState]
}

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor, TaskState]) Enable(ctx context.Context) error {

	l.logger.Info("Enabling connector")
	err := l.store.Enable(ctx, l.loader.Name())
	if err != nil {
		return err
	}

	return nil
}

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor, TaskState]) ReadConfig(ctx context.Context) (*ConnectorConfig, error) {

	var config ConnectorConfig
	err := l.store.ReadConfig(ctx, l.loader.Name(), &config)
	if err != nil {
		return &config, err
	}

	config = l.loader.ApplyDefaults(config)

	return &config, nil
}

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor, TaskState]) load(config ConnectorConfig) {
	l.connector = l.loader.Load(l.logger, config)
	l.scheduler = l.schedulerFactory.Make(l.connector, l.loader.AllowTasks())
}

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor, TaskState]) Install(ctx context.Context, config ConnectorConfig) (err error) {

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

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor, TaskState]) Uninstall(ctx context.Context) error {

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

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor, TaskState]) Restore(ctx context.Context) error {
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

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor, TaskState]) Disable(ctx context.Context) error {
	l.logger.Info("Disabling connector")

	return l.store.Disable(ctx, l.loader.Name())
}

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor, TaskState]) IsEnabled(ctx context.Context) (bool, error) {
	return l.store.IsEnabled(ctx, l.loader.Name())
}

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor, TaskState]) IsInstalled(ctx context.Context) (bool, error) {
	return l.store.IsInstalled(ctx, l.loader.Name())
}

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor, TaskState]) ListTasksStates(ctx context.Context) ([]payments.TaskState[TaskDescriptor, TaskState], error) {
	return l.scheduler.ListTasks(ctx)
}

func (l ConnectorManager[Config, TaskDescriptor, TaskState]) ReadTaskState(ctx context.Context, descriptor TaskDescriptor) (*payments.TaskState[TaskDescriptor, TaskState], error) {
	return l.scheduler.ReadTask(ctx, descriptor)
}

func (l *ConnectorManager[ConnectorConfig, TaskDescriptor, TaskState]) Reset(ctx context.Context) error {
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
	TaskState any,
](
	logger sharedlogging.Logger,
	store ConnectorStore,
	loader Loader[ConnectorConfig, TaskDescriptor, TaskState],
	schedulerFactory TaskSchedulerFactory[TaskDescriptor, TaskState],
) *ConnectorManager[ConnectorConfig, TaskDescriptor, TaskState] {
	return &ConnectorManager[ConnectorConfig, TaskDescriptor, TaskState]{
		logger: logger.WithFields(map[string]interface{}{
			"component": "connector-manager",
			"provider":  loader.Name(),
		}),
		store:            store,
		loader:           loader,
		schedulerFactory: schedulerFactory,
	}
}
