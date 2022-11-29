package integration

import (
	"context"
	"testing"

	"github.com/formancehq/payments/internal/pkg/payments"
	"github.com/formancehq/payments/internal/pkg/task"

	"github.com/formancehq/go-libs/sharedlogging"
	"github.com/formancehq/go-libs/sharedlogging/sharedlogginglogrus"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func ChanClosed[T any](ch chan T) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

type testContext[ConnectorConfig payments.ConnectorConfigObject,
	TaskDescriptor payments.TaskDescriptor] struct {
	manager        *ConnectorManager[ConnectorConfig, TaskDescriptor]
	taskStore      task.Store[TaskDescriptor]
	connectorStore ConnectorStore
	loader         Loader[ConnectorConfig, TaskDescriptor]
	provider       string
}

func withManager[ConnectorConfig payments.ConnectorConfigObject,
	TaskDescriptor payments.TaskDescriptor](builder *ConnectorBuilder[TaskDescriptor],
	callback func(ctx *testContext[ConnectorConfig, TaskDescriptor]),
) {
	l := logrus.New()
	if testing.Verbose() {
		l.SetLevel(logrus.DebugLevel)
	}

	logger := sharedlogginglogrus.New(l)
	taskStore := task.NewInMemoryStore[TaskDescriptor]()
	managerStore := NewInMemoryStore()
	provider := uuid.New()
	schedulerFactory := TaskSchedulerFactoryFn[TaskDescriptor](func(resolver task.Resolver[TaskDescriptor],
		maxTasks int,
	) *task.DefaultTaskScheduler[TaskDescriptor] {
		return task.NewDefaultScheduler[TaskDescriptor](provider, logger, taskStore,
			task.DefaultContainerFactory, resolver, maxTasks)
	})

	loader := NewLoaderBuilder[ConnectorConfig, TaskDescriptor](provider).
		WithLoad(func(logger sharedlogging.Logger, config ConnectorConfig) Connector[TaskDescriptor] {
			return builder.Build()
		}).
		WithAllowedTasks(1).
		Build()
	manager := NewConnectorManager[ConnectorConfig, TaskDescriptor](logger, managerStore, loader,
		schedulerFactory)

	defer func() {
		_ = manager.Uninstall(context.Background())
	}()

	callback(&testContext[ConnectorConfig, TaskDescriptor]{
		manager:        manager,
		taskStore:      taskStore,
		connectorStore: managerStore,
		loader:         loader,
		provider:       provider,
	})
}

func TestInstallConnector(t *testing.T) {
	t.Parallel()

	installed := make(chan struct{})
	builder := NewConnectorBuilder[any]().
		WithInstall(func(ctx task.ConnectorContext[any]) error {
			close(installed)

			return nil
		})
	withManager(builder, func(tc *testContext[payments.EmptyConnectorConfig, any]) {
		err := tc.manager.Install(context.Background(), payments.EmptyConnectorConfig{})
		require.NoError(t, err)
		require.True(t, ChanClosed(installed))

		err = tc.manager.Install(context.Background(), payments.EmptyConnectorConfig{})
		require.Equal(t, ErrAlreadyInstalled, err)
	})
}

func TestUninstallConnector(t *testing.T) {
	t.Parallel()

	uninstalled := make(chan struct{})
	taskTerminated := make(chan struct{})
	taskStarted := make(chan struct{})
	builder := NewConnectorBuilder[any]().
		WithResolve(func(name any) task.Task {
			return func(ctx context.Context, stopChan task.StopChan) {
				close(taskStarted)
				defer close(taskTerminated)
				select {
				case flag := <-stopChan:
					flag <- struct{}{}
				case <-ctx.Done():
				}
			}
		}).
		WithInstall(func(ctx task.ConnectorContext[any]) error {
			return ctx.Scheduler().Schedule(uuid.New(), false)
		}).
		WithUninstall(func(ctx context.Context) error {
			close(uninstalled)

			return nil
		})
	withManager(builder, func(tc *testContext[payments.EmptyConnectorConfig, any]) {
		err := tc.manager.Install(context.Background(), payments.EmptyConnectorConfig{})
		require.NoError(t, err)
		<-taskStarted
		require.NoError(t, tc.manager.Uninstall(context.Background()))
		require.True(t, ChanClosed(uninstalled))
		// TODO: We need to give a chance to the connector to properly stop execution
		require.True(t, ChanClosed(taskTerminated))

		isInstalled, err := tc.manager.IsInstalled(context.Background())
		require.NoError(t, err)
		require.False(t, isInstalled)
	})
}

func TestDisableConnector(t *testing.T) {
	t.Parallel()

	uninstalled := make(chan struct{})
	builder := NewConnectorBuilder[any]().
		WithUninstall(func(ctx context.Context) error {
			close(uninstalled)

			return nil
		})
	withManager[payments.EmptyConnectorConfig, any](builder, func(tc *testContext[payments.EmptyConnectorConfig, any]) {
		err := tc.manager.Install(context.Background(), payments.EmptyConnectorConfig{})
		require.NoError(t, err)

		enabled, err := tc.manager.IsEnabled(context.Background())
		require.NoError(t, err)
		require.True(t, enabled)

		require.NoError(t, tc.manager.Disable(context.Background()))
		enabled, err = tc.manager.IsEnabled(context.Background())
		require.NoError(t, err)
		require.False(t, enabled)
	})
}

func TestEnableConnector(t *testing.T) {
	t.Parallel()

	builder := NewConnectorBuilder[any]()
	withManager[payments.EmptyConnectorConfig, any](builder, func(tc *testContext[payments.EmptyConnectorConfig, any]) {
		err := tc.connectorStore.Enable(context.Background(), tc.loader.Name())
		require.NoError(t, err)

		err = tc.manager.Install(context.Background(), payments.EmptyConnectorConfig{})
		require.NoError(t, err)
	})
}

func TestRestoreEnabledConnector(t *testing.T) {
	t.Parallel()

	builder := NewConnectorBuilder[any]()
	withManager(builder, func(tc *testContext[payments.EmptyConnectorConfig, any]) {
		err := tc.connectorStore.Install(context.Background(), tc.loader.Name(), payments.EmptyConnectorConfig{})
		require.NoError(t, err)

		err = tc.manager.Restore(context.Background())
		require.NoError(t, err)
		require.NotNil(t, tc.manager.connector)
	})
}

func TestRestoreNotInstalledConnector(t *testing.T) {
	t.Parallel()

	builder := NewConnectorBuilder[any]()
	withManager(builder, func(tc *testContext[payments.EmptyConnectorConfig, any]) {
		err := tc.manager.Restore(context.Background())
		require.Equal(t, ErrNotInstalled, err)
	})
}
