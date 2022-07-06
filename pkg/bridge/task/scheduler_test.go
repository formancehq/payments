package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/numary/go-libs/sharedlogging/sharedloggingtesting"
	payments "github.com/numary/payments/pkg"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TaskTerminatedWithStatus[TaskDescriptor payments.TaskDescriptor](store *inMemoryStore[TaskDescriptor], provider string, descriptor TaskDescriptor, expectedStatus payments.TaskStatus, errString string) func() bool {
	return func() bool {
		status, err, ok := store.Result(provider, descriptor)
		if !ok {
			return false
		}
		if err != errString {
			return false
		}
		return status == expectedStatus
	}
}

func TaskTerminated[TaskDescriptor payments.TaskDescriptor](store *inMemoryStore[TaskDescriptor], provider string, descriptor TaskDescriptor) func() bool {
	return TaskTerminatedWithStatus(store, provider, descriptor, payments.TaskStatusTerminated, "")
}

func TaskFailed[TaskDescriptor payments.TaskDescriptor](store *inMemoryStore[TaskDescriptor], provider string, descriptor TaskDescriptor, errStr string) func() bool {
	return TaskTerminatedWithStatus(store, provider, descriptor, payments.TaskStatusFailed, errStr)
}

func TaskPending[TaskDescriptor payments.TaskDescriptor](store *inMemoryStore[TaskDescriptor], provider string, descriptor TaskDescriptor) func() bool {
	return TaskTerminatedWithStatus(store, provider, descriptor, payments.TaskStatusPending, "")
}

func TaskActive[TaskDescriptor payments.TaskDescriptor](store *inMemoryStore[TaskDescriptor], provider string, descriptor TaskDescriptor) func() bool {
	return TaskTerminatedWithStatus(store, provider, descriptor, payments.TaskStatusActive, "")
}

func TestTaskScheduler(t *testing.T) {
	logger := sharedloggingtesting.Logger()

	t.Run("Nominal", func(t *testing.T) {
		store := NewInMemoryStore[string]()
		provider := uuid.New()
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
		store := NewInMemoryStore[string]()
		provider := uuid.New()
		scheduler := NewDefaultScheduler[string](provider, logger, store, DefaultContainerFactory, ResolverFn[string](func(descriptor string) Task {
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
		provider := uuid.New()
		store := NewInMemoryStore[string]()
		scheduler := NewDefaultScheduler[string](provider, logger, store, DefaultContainerFactory, ResolverFn[string](func(descriptor string) Task {
			return func() error {
				return errors.New("test")
			}
		}), 1)

		descriptor := uuid.New()
		err := scheduler.Schedule(descriptor, false)
		require.NoError(t, err)
		require.Eventually(t, TaskFailed(store, provider, descriptor, "test"), time.Second, 100*time.Millisecond)
	})

	t.Run("Pending", func(t *testing.T) {
		provider := uuid.New()
		store := NewInMemoryStore[string]()
		descriptor1 := uuid.New()
		descriptor2 := uuid.New()

		task1Terminated := make(chan struct{})
		task2Terminated := make(chan struct{})

		scheduler := NewDefaultScheduler[string](provider, logger, store, DefaultContainerFactory, ResolverFn[string](func(descriptor string) Task {
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
		provider := uuid.New()
		store := NewInMemoryStore[string]()
		scheduler := NewDefaultScheduler[string](provider, logger, store, DefaultContainerFactory, ResolverFn[string](func(descriptor string) Task {
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
