package task

import (
	"context"
	"errors"
	"github.com/numary/go-libs/sharedlogging/sharedloggingtesting"
	payments "github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge/ingestion"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TaskTerminatedWithStatus[TaskDescriptor payments.TaskDescriptor, TaskState any](store *inMemoryStore[TaskDescriptor, TaskState], provider string, descriptor TaskDescriptor, expectedStatus payments.TaskStatus, errString string) func() bool {
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

func TaskTerminated[TaskDescriptor payments.TaskDescriptor, TaskState any](store *inMemoryStore[TaskDescriptor, TaskState], provider string, descriptor TaskDescriptor) func() bool {
	return TaskTerminatedWithStatus(store, provider, descriptor, payments.TaskStatusTerminated, "")
}

func TaskFailed[TaskDescriptor payments.TaskDescriptor, TaskState any](store *inMemoryStore[TaskDescriptor, TaskState], provider string, descriptor TaskDescriptor, errStr string) func() bool {
	return TaskTerminatedWithStatus(store, provider, descriptor, payments.TaskStatusFailed, errStr)
}

func TaskPending[TaskDescriptor payments.TaskDescriptor, TaskState any](store *inMemoryStore[TaskDescriptor, TaskState], provider string, descriptor TaskDescriptor) func() bool {
	return TaskTerminatedWithStatus(store, provider, descriptor, payments.TaskStatusPending, "")
}

func TaskActive[TaskDescriptor payments.TaskDescriptor, TaskState any](store *inMemoryStore[TaskDescriptor, TaskState], provider string, descriptor TaskDescriptor) func() bool {
	return TaskTerminatedWithStatus(store, provider, descriptor, payments.TaskStatusActive, "")
}

func TestTaskScheduler(t *testing.T) {
	type State struct {
		Counter int `json:"counter"`
	}

	logger := sharedloggingtesting.Logger()

	t.Run("Nominal", func(t *testing.T) {
		store := NewInMemoryStore[string, any]()
		provider := uuid.New()
		done := make(chan struct{})

		scheduler := NewDefaultScheduler[string, any](provider, logger, store,
			NoOpIngesterFactory, ResolverFn[string, any](func(descriptor string) Task[string, any] {
				return NewFunctionTask[string, any](RunnerFn[string, any](func(ctx Context[string, any]) error {
					select {
					case <-ctx.Context().Done():
						return ctx.Context().Err()
					case <-done:
						return nil
					}
				}))
			}), 1)

		descriptor := uuid.New()
		err := scheduler.Schedule(descriptor)
		require.NoError(t, err)

		require.Eventually(t, TaskActive(store, provider, descriptor), time.Second, 100*time.Millisecond)
		close(done)
		require.Eventually(t, TaskTerminated(store, provider, descriptor), time.Second, 100*time.Millisecond)
	})

	t.Run("Duplicate task", func(t *testing.T) {
		store := NewInMemoryStore[string, any]()
		provider := uuid.New()
		scheduler := NewDefaultScheduler[string, any](provider, logger, store, NoOpIngesterFactory, ResolverFn[string, any](func(descriptor string) Task[string, any] {
			return NewFunctionTask[string, any](RunnerFn[string, any](func(ctx Context[string, any]) error {
				select {
				case <-ctx.Context().Done():
					return ctx.Context().Err()
				}
			}))
		}), 1)

		descriptor := uuid.New()
		err := scheduler.Schedule(descriptor)
		require.NoError(t, err)
		require.Eventually(t, TaskActive(store, provider, descriptor), time.Second, 100*time.Millisecond)

		err = scheduler.Schedule(descriptor)
		require.Equal(t, ErrAlreadyScheduled, err)
	})

	t.Run("Ingest", func(t *testing.T) {
		provider := uuid.New()
		store := NewInMemoryStore[string, any]()
		scheduler := NewDefaultScheduler[string, any](provider, logger, store, NoOpIngesterFactory, ResolverFn[string, any](func(descriptor string) Task[string, any] {
			return NewFunctionTask[string, any](RunnerFn[string, any](func(ctx Context[string, any]) error {
				return ctx.Ingester().Ingest(ctx.Context(), ingestion.Batch{
					{
						Referenced: payments.Referenced{
							Reference: "p1",
							Type:      payments.TypePayIn,
						},
					},
				}, State{
					Counter: 2,
				})
			}))
		}), 1)

		descriptor := uuid.New()
		err := scheduler.Schedule(descriptor)
		require.NoError(t, err)
		require.Eventually(t, TaskTerminated(store, provider, descriptor), time.Second, 100*time.Millisecond)
	})

	t.Run("Error", func(t *testing.T) {
		provider := uuid.New()
		store := NewInMemoryStore[string, any]()
		scheduler := NewDefaultScheduler[string, any](provider, logger, store, NoOpIngesterFactory, ResolverFn[string, any](func(descriptor string) Task[string, any] {
			return NewFunctionTask[string, any](RunnerFn[string, any](func(ctx Context[string, any]) error {
				return errors.New("test")
			}))
		}), 1)

		descriptor := uuid.New()
		err := scheduler.Schedule(descriptor)
		require.NoError(t, err)
		require.Eventually(t, TaskFailed(store, provider, descriptor, "test"), time.Second, 100*time.Millisecond)
	})

	t.Run("Pending", func(t *testing.T) {
		provider := uuid.New()
		store := NewInMemoryStore[string, any]()
		descriptor1 := uuid.New()
		descriptor2 := uuid.New()

		task1Terminated := make(chan struct{})
		task2Terminated := make(chan struct{})

		scheduler := NewDefaultScheduler[string, any](provider, logger, store, NoOpIngesterFactory, ResolverFn[string, any](func(descriptor string) Task[string, any] {
			switch descriptor {
			case descriptor1:
				return NewFunctionTask[string, any](RunnerFn[string, any](func(ctx Context[string, any]) error {
					select {
					case <-task1Terminated:
						return nil
					case <-ctx.Context().Done():
						return ctx.Context().Err()
					}
				}))
			case descriptor2:
				return NewFunctionTask[string, any](RunnerFn[string, any](func(ctx Context[string, any]) error {
					select {
					case <-task2Terminated:
						return nil
					case <-ctx.Context().Done():
						return ctx.Context().Err()
					}
				}))
			}
			panic("unknown descriptor")
		}), 1)

		require.NoError(t, scheduler.Schedule(descriptor1))
		require.NoError(t, scheduler.Schedule(descriptor2))
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
		store := NewInMemoryStore[string, any]()
		scheduler := NewDefaultScheduler[string, any](provider, logger, store, NoOpIngesterFactory, ResolverFn[string, any](func(descriptor string) Task[string, any] {
			switch descriptor {
			case "main":
				return NewFunctionTask[string, any](RunnerFn[string, any](func(ctx Context[string, any]) error {
					select {
					case <-ctx.Context().Done():
						require.NoError(t, ctx.Scheduler().Schedule("worker"))
						return nil
					}
				}))
			default:
				panic("should not be called")
			}
		}), 1)

		require.NoError(t, scheduler.Schedule("main"))
		require.Eventually(t, TaskActive(store, provider, "main"), time.Second, 100*time.Millisecond)
		require.NoError(t, scheduler.Shutdown(context.Background()))
		require.Eventually(t, TaskTerminated(store, provider, "main"), time.Second, 100*time.Millisecond)
		require.Eventually(t, TaskPending(store, provider, "worker"), time.Second, 100*time.Millisecond)
	})
}
