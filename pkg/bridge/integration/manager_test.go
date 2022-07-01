package integration

import (
	"context"
	"testing"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/go-libs/sharedlogging/sharedloggingtesting"
	payments "github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge/task"
	"github.com/pborman/uuid"
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

type testContext[ConnectorConfig payments.ConnectorConfigObject, TaskDescriptor payments.TaskDescriptor, TaskState any] struct {
	manager        *ConnectorManager[ConnectorConfig, TaskDescriptor, TaskState]
	taskStore      task.Store[TaskDescriptor, TaskState]
	connectorStore ConnectorStore
	loader         Loader[ConnectorConfig, TaskDescriptor, TaskState]
	provider       string
}

func withManager[ConnectorConfig payments.ConnectorConfigObject, TaskDescriptor payments.TaskDescriptor, TaskState any](builder *ConnectorBuilder[TaskDescriptor, TaskState], callback func(ctx *testContext[ConnectorConfig, TaskDescriptor, TaskState])) {
	logger := sharedloggingtesting.Logger()
	taskStore := task.NewInMemoryStore[TaskDescriptor, TaskState]()
	managerStore := NewInMemoryStore()
	provider := uuid.New()
	schedulerFactory := TaskSchedulerFactoryFn[TaskDescriptor, TaskState](func(resolver task.Resolver[TaskDescriptor, TaskState], maxTasks int) *task.DefaultTaskScheduler[TaskDescriptor, TaskState] {
		return task.NewDefaultScheduler[TaskDescriptor, TaskState](provider, logger, taskStore, task.NoOpIngesterFactory, resolver, maxTasks)
	})

	loader := NewLoaderBuilder[ConnectorConfig, TaskDescriptor, TaskState](provider).
		WithLoad(func(logger sharedlogging.Logger, config ConnectorConfig) Connector[TaskDescriptor, TaskState] {
			return builder.Build()
		}).
		WithAllowedTasks(1).
		Build()
	manager := NewConnectorManager[ConnectorConfig, TaskDescriptor, TaskState](logger, managerStore, loader, schedulerFactory)
	defer manager.Uninstall(context.Background())

	callback(&testContext[ConnectorConfig, TaskDescriptor, TaskState]{
		manager:        manager,
		taskStore:      taskStore,
		connectorStore: managerStore,
		loader:         loader,
		provider:       provider,
	})
}

func TestInstallConnector(t *testing.T) {
	installed := make(chan struct{})
	builder := NewConnectorBuilder[any, any]().
		WithInstall(func(ctx task.ConnectorContext[any]) error {
			close(installed)
			return nil
		})
	withManager(builder, func(tc *testContext[payments.EmptyConnectorConfig, any, any]) {
		err := tc.manager.Install(context.Background(), payments.EmptyConnectorConfig{})
		require.NoError(t, err)
		require.True(t, ChanClosed(installed))

		err = tc.manager.Install(context.Background(), payments.EmptyConnectorConfig{})
		require.Equal(t, ErrAlreadyInstalled, err)
	})
}

func TestUninstallConnector(t *testing.T) {
	uninstalled := make(chan struct{})
	taskTerminated := make(chan struct{})
	taskStarted := make(chan struct{})
	builder := NewConnectorBuilder[any, any]().
		WithResolve(func(name any) task.Task[any, any] {
			return task.NewFunctionTask[any, any](task.RunnerFn[any, any](func(ctx task.Context[any, any]) error {
				close(taskStarted)
				defer close(taskTerminated)
				select {
				case <-ctx.Context().Done():
				}
				return nil
			}))
		}).
		WithInstall(func(ctx task.ConnectorContext[any]) error {
			return ctx.Scheduler().Schedule(uuid.New(), false)
		}).
		WithUninstall(func(ctx context.Context) error {
			close(uninstalled)
			return nil
		})
	withManager(builder, func(tc *testContext[payments.EmptyConnectorConfig, any, any]) {
		err := tc.manager.Install(context.Background(), payments.EmptyConnectorConfig{})
		require.NoError(t, err)
		<-taskStarted
		require.NoError(t, tc.manager.Uninstall(context.Background()))
		require.True(t, ChanClosed(uninstalled))
		require.True(t, ChanClosed(taskTerminated))

		isInstalled, err := tc.manager.IsInstalled(context.Background())
		require.NoError(t, err)
		require.False(t, isInstalled)
	})
}

func TestDisableConnector(t *testing.T) {
	uninstalled := make(chan struct{})
	builder := NewConnectorBuilder[any, any]().
		WithUninstall(func(ctx context.Context) error {
			close(uninstalled)
			return nil
		})
	withManager[payments.EmptyConnectorConfig, any, any](builder, func(tc *testContext[payments.EmptyConnectorConfig, any, any]) {
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
	builder := NewConnectorBuilder[any, any]()
	withManager[payments.EmptyConnectorConfig, any](builder, func(tc *testContext[payments.EmptyConnectorConfig, any, any]) {
		err := tc.connectorStore.Enable(context.Background(), tc.loader.Name())
		require.NoError(t, err)

		err = tc.manager.Install(context.Background(), payments.EmptyConnectorConfig{})
		require.NoError(t, err)
	})
}

func TestRestoreEnabledConnector(t *testing.T) {
	builder := NewConnectorBuilder[any, any]()
	withManager(builder, func(tc *testContext[payments.EmptyConnectorConfig, any, any]) {
		err := tc.connectorStore.Install(context.Background(), tc.loader.Name(), payments.EmptyConnectorConfig{})
		require.NoError(t, err)

		err = tc.manager.Restore(context.Background())
		require.NoError(t, err)
		require.NotNil(t, tc.manager.connector)
	})
}

func TestRestoreNotInstalledConnector(t *testing.T) {
	builder := NewConnectorBuilder[any, any]()
	withManager(builder, func(tc *testContext[payments.EmptyConnectorConfig, any, any]) {
		err := tc.manager.Restore(context.Background())
		require.Equal(t, ErrNotInstalled, err)
	})
}
