package task

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/app/models"

	"github.com/formancehq/payments/internal/app/payments"

	"github.com/formancehq/go-libs/sharedlogging/sharedlogginglogrus"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TaskTerminatedWithStatus[TaskDescriptor payments.TaskDescriptor](store *InMemoryStore,
	provider models.ConnectorProvider, descriptor TaskDescriptor, expectedStatus models.TaskStatus, errString string,
) func() bool {
	return func() bool {
		taskDescriptor, err := json.Marshal(descriptor)
		if err != nil {
			return false
		}

		status, resultErr, ok := store.Result(provider, taskDescriptor)
		if !ok {
			return false
		}

		if resultErr != errString {
			return false
		}

		return status == expectedStatus
	}
}

func TaskTerminated[TaskDescriptor payments.TaskDescriptor](store *InMemoryStore,
	provider models.ConnectorProvider, descriptor TaskDescriptor,
) func() bool {
	return TaskTerminatedWithStatus(store, provider, descriptor, models.TaskStatusTerminated, "")
}

func TaskFailed[TaskDescriptor payments.TaskDescriptor](store *InMemoryStore,
	provider models.ConnectorProvider, descriptor TaskDescriptor, errStr string,
) func() bool {
	return TaskTerminatedWithStatus(store, provider, descriptor, models.TaskStatusFailed, errStr)
}

func TaskPending[TaskDescriptor payments.TaskDescriptor](store *InMemoryStore,
	provider models.ConnectorProvider, descriptor TaskDescriptor,
) func() bool {
	return TaskTerminatedWithStatus(store, provider, descriptor, models.TaskStatusPending, "")
}

func TaskActive[TaskDescriptor payments.TaskDescriptor](store *InMemoryStore,
	provider models.ConnectorProvider, descriptor TaskDescriptor,
) func() bool {
	return TaskTerminatedWithStatus(store, provider, descriptor, models.TaskStatusActive, "")
}

func TestTaskScheduler(t *testing.T) {
	t.Parallel()

	l := logrus.New()
	if testing.Verbose() {
		l.SetLevel(logrus.DebugLevel)
	}

	logger := sharedlogginglogrus.New(l)

	t.Run("Nominal", func(t *testing.T) {
		t.Parallel()

		store := NewInMemoryStore()
		provider := models.ConnectorProvider(uuid.New())
		done := make(chan struct{})

		scheduler := NewDefaultScheduler[string](provider, logger, store,
			DefaultContainerFactory, ResolverFn[string](func(descriptor string) Task {
				return func(ctx context.Context) error {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-done:
						return nil
					}
				}
			}), 1)

		descriptor := uuid.New()
		err := scheduler.Schedule(descriptor, false)
		require.NoError(t, err)

		require.Eventually(t, TaskActive(store, provider, descriptor), time.Second, 100*time.Millisecond)
		close(done)
		require.Eventually(t, TaskTerminated(store, provider, descriptor), time.Second, 100*time.Millisecond)
	})

	t.Run("Duplicate task", func(t *testing.T) {
		t.Parallel()

		store := NewInMemoryStore()
		provider := models.ConnectorProvider(uuid.New())
		scheduler := NewDefaultScheduler[string](provider, logger, store, DefaultContainerFactory,
			ResolverFn[string](func(descriptor string) Task {
				return func(ctx context.Context) error {
					<-ctx.Done()

					return ctx.Err()
				}
			}), 1)

		descriptor := uuid.New()
		err := scheduler.Schedule(descriptor, false)
		require.NoError(t, err)
		require.Eventually(t, TaskActive(store, provider, descriptor), time.Second, 100*time.Millisecond)

		err = scheduler.Schedule(descriptor, false)
		require.Equal(t, ErrAlreadyScheduled, err)
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		provider := models.ConnectorProvider(uuid.New())
		store := NewInMemoryStore()
		scheduler := NewDefaultScheduler[string](provider, logger, store, DefaultContainerFactory,
			ResolverFn[string](func(descriptor string) Task {
				return func() error {
					return errors.New("test")
				}
			}), 1)

		descriptor := uuid.New()
		err := scheduler.Schedule(descriptor, false)
		require.NoError(t, err)
		require.Eventually(t, TaskFailed(store, provider, descriptor, "test"), time.Second,
			100*time.Millisecond)
	})

	t.Run("Pending", func(t *testing.T) {
		t.Parallel()

		provider := models.ConnectorProvider(uuid.New())
		store := NewInMemoryStore()
		descriptor1 := uuid.New()
		descriptor2 := uuid.New()

		task1Terminated := make(chan struct{})
		task2Terminated := make(chan struct{})

		scheduler := NewDefaultScheduler[string](provider, logger, store, DefaultContainerFactory,
			ResolverFn[string](func(descriptor string) Task {
				switch descriptor {
				case descriptor1:
					return func(ctx context.Context) error {
						select {
						case <-task1Terminated:
							return nil
						case <-ctx.Done():
							return ctx.Err()
						}
					}
				case descriptor2:
					return func(ctx context.Context) error {
						select {
						case <-task2Terminated:
							return nil
						case <-ctx.Done():
							return ctx.Err()
						}
					}
				}
				panic("unknown descriptor")
			}), 1)

		require.NoError(t, scheduler.Schedule(descriptor1, false))
		require.NoError(t, scheduler.Schedule(descriptor2, false))
		require.Eventually(t, TaskActive(store, provider, descriptor1), time.Second, 100*time.Millisecond)
		require.Eventually(t, TaskPending(store, provider, descriptor2), time.Second, 100*time.Millisecond)
		close(task1Terminated)
		require.Eventually(t, TaskTerminated(store, provider, descriptor1), time.Second, 100*time.Millisecond)
		require.Eventually(t, TaskActive(store, provider, descriptor2), time.Second, 100*time.Millisecond)
		close(task2Terminated)
		require.Eventually(t, TaskTerminated(store, provider, descriptor2), time.Second, 100*time.Millisecond)
	})

	t.Run("Stop scheduler", func(t *testing.T) {
		t.Parallel()

		provider := models.ConnectorProvider(uuid.New())
		store := NewInMemoryStore()
		scheduler := NewDefaultScheduler[string](provider, logger, store, DefaultContainerFactory,
			ResolverFn[string](func(descriptor string) Task {
				switch descriptor {
				case "main":
					return func(ctx context.Context, scheduler Scheduler[string]) {
						<-ctx.Done()
						require.NoError(t, scheduler.Schedule("worker", false))
					}
				default:
					panic("should not be called")
				}
			}), 1)

		require.NoError(t, scheduler.Schedule("main", false))
		require.Eventually(t, TaskActive(store, provider, "main"), time.Second, 100*time.Millisecond)
		require.NoError(t, scheduler.Shutdown(context.Background()))
		require.Eventually(t, TaskTerminated(store, provider, "main"), time.Second, 100*time.Millisecond)
		require.Eventually(t, TaskPending(store, provider, "worker"), time.Second, 100*time.Millisecond)
	})
}
