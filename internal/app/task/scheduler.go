package task

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/formancehq/payments/internal/app/storage"

	"github.com/formancehq/payments/internal/app/models"

	"github.com/formancehq/go-libs/sharedlogging"
	"github.com/formancehq/payments/internal/app/payments"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrAlreadyScheduled = errors.New("already scheduled")
	ErrUnableToResolve  = errors.New("unable to resolve task")
)

type Scheduler[TaskDescriptor payments.TaskDescriptor] interface {
	Schedule(p TaskDescriptor, restart bool) error
}

type taskHolder[TaskDescriptor payments.TaskDescriptor] struct {
	descriptor json.RawMessage
	cancel     func()
	logger     sharedlogging.Logger
	stopChan   StopChan
}

type DefaultTaskScheduler[TaskDescriptor payments.TaskDescriptor] struct {
	provider         models.ConnectorProvider
	logger           sharedlogging.Logger
	store            Repository
	containerFactory ContainerFactory
	tasks            map[string]*taskHolder[TaskDescriptor]
	mu               sync.Mutex
	maxTasks         int
	resolver         Resolver[TaskDescriptor]
	stopped          bool
}

func (s *DefaultTaskScheduler[TaskDescriptor]) ListTasks(ctx context.Context,
) ([]models.Task, error) {
	return s.store.ListTasks(ctx, s.provider)
}

func (s *DefaultTaskScheduler[TaskDescriptor]) ReadTask(ctx context.Context, taskID uuid.UUID) (*models.Task, error) {
	return s.store.GetTask(ctx, taskID)
}

func (s *DefaultTaskScheduler[TaskDescriptor]) ReadTaskByDescriptor(ctx context.Context,
	descriptor TaskDescriptor,
) (*models.Task, error) {
	taskDescriptor, err := json.Marshal(descriptor)
	if err != nil {
		return nil, err
	}

	return s.store.GetTaskByDescriptor(ctx, s.provider, taskDescriptor)
}

func (s *DefaultTaskScheduler[TaskDescriptor]) Schedule(descriptor TaskDescriptor, restart bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	taskID := payments.IDFromDescriptor(descriptor)
	if _, ok := s.tasks[taskID]; ok {
		return ErrAlreadyScheduled
	}

	if !restart {
		_, err := s.ReadTaskByDescriptor(context.Background(), descriptor)
		if err == nil {
			return nil
		}
	}

	if s.maxTasks != 0 && len(s.tasks) >= s.maxTasks || s.stopped {
		err := s.stackTask(descriptor)
		if err != nil {
			return errors.Wrap(err, "stacking task")
		}

		return nil
	}

	taskDescriptor, err := json.Marshal(descriptor)
	if err != nil {
		return err
	}

	if err := s.startTask(taskDescriptor); err != nil {
		return errors.Wrap(err, "starting task")
	}

	return nil
}

func (s *DefaultTaskScheduler[TaskDescriptor]) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.stopped = true
	s.mu.Unlock()

	s.logger.Infof("Stopping scheduler...")

	for name, task := range s.tasks {
		task.logger.Debugf("Stopping task")

		if task.stopChan != nil {
			errCh := make(chan struct{})
			task.stopChan <- errCh
			select {
			case <-errCh:
			case <-time.After(time.Second): // TODO: Make configurable
				task.logger.Debugf("Stopping using stop chan timeout, canceling context")
				task.cancel()
			}
		} else {
			task.cancel()
		}

		delete(s.tasks, name)
	}

	return nil
}

func (s *DefaultTaskScheduler[TaskDescriptor]) Restore(ctx context.Context) error {
	tasks, err := s.store.ListTasksByStatus(ctx, s.provider, models.TaskStatusActive)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		err = s.startTask(task.Descriptor)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *DefaultTaskScheduler[TaskDescriptor]) registerTaskError(ctx context.Context,
	holder *taskHolder[TaskDescriptor], taskErr any,
) {
	var taskError string

	switch v := taskErr.(type) {
	case error:
		taskError = v.Error()
	default:
		taskError = fmt.Sprintf("%s", v)
	}

	holder.logger.Errorf("Task terminated with error: %s", taskErr)

	err := s.store.UpdateTaskStatus(ctx, s.provider, holder.descriptor, models.TaskStatusFailed, taskError)
	if err != nil {
		holder.logger.Error("Error updating task status: %s", taskError)
	}
}

func (s *DefaultTaskScheduler[TaskDescriptor]) deleteTask(holder *taskHolder[TaskDescriptor]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tasks, payments.IDFromDescriptor(holder.descriptor))

	if s.stopped {
		return
	}

	oldestPendingTask, err := s.store.ReadOldestPendingTask(context.Background(), s.provider)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return
		}

		sharedlogging.Error(err)

		return
	}

	var descriptor TaskDescriptor

	err = json.Unmarshal(oldestPendingTask.Descriptor, &descriptor)
	if err != nil {
		sharedlogging.Error(err)

		return
	}

	p := s.resolver.Resolve(descriptor)
	if p == nil {
		sharedlogging.Errorf("unable to resolve task")

		return
	}

	err = s.startTask(oldestPendingTask.Descriptor)
	if err != nil {
		sharedlogging.Error(err)
	}
}

type StopChan chan chan struct{}

func (s *DefaultTaskScheduler[TaskDescriptor]) startTask(descriptor json.RawMessage) error {
	task, err := s.store.FindAndUpsertTask(context.Background(), s.provider, descriptor,
		models.TaskStatusActive, "")
	if err != nil {
		return errors.Wrap(err, "finding task and update")
	}

	logger := s.logger.WithFields(map[string]interface{}{
		"task-id": task.ID,
	})

	var taskDescriptor TaskDescriptor

	err = json.Unmarshal(task.Descriptor, &taskDescriptor)
	if err != nil {
		return err
	}

	taskResolver := s.resolver.Resolve(taskDescriptor)
	if taskResolver == nil {
		return ErrUnableToResolve
	}

	ctx, cancel := context.WithCancel(context.Background())
	ctx, span := otel.Tracer("com.formance.payments").Start(ctx, "Task", trace.WithAttributes(
		attribute.Stringer("id", task.ID),
		attribute.Stringer("connector", s.provider),
	))

	holder := &taskHolder[TaskDescriptor]{
		cancel:     cancel,
		logger:     logger,
		descriptor: descriptor,
	}

	container, err := s.containerFactory.Create(ctx, descriptor)
	if err != nil {
		// TODO: Handle error
		panic(err)
	}

	err = container.Provide(func() context.Context {
		return ctx
	})
	if err != nil {
		panic(err)
	}

	err = container.Provide(func() Scheduler[TaskDescriptor] {
		return s
	})
	if err != nil {
		panic(err)
	}

	err = container.Provide(func() StopChan {
		s.mu.Lock()
		defer s.mu.Unlock()

		holder.stopChan = make(StopChan, 1)

		return holder.stopChan
	})
	if err != nil {
		panic(err)
	}

	err = container.Provide(func() sharedlogging.Logger {
		return s.logger
	})
	if err != nil {
		panic(err)
	}

	err = container.Provide(func() StateResolver {
		return StateResolverFn(func(ctx context.Context, v any) error {
			if task.State == nil || len(task.State) == 0 {
				return nil
			}

			return json.Unmarshal(task.State, v)
		})
	})
	if err != nil {
		panic(err)
	}

	s.tasks[payments.IDFromDescriptor(descriptor)] = holder

	go func() {
		logger.Infof("Starting task...")

		defer func() {
			defer span.End()
			defer s.deleteTask(holder)

			if e := recover(); e != nil {
				s.registerTaskError(ctx, holder, e)
				debug.PrintStack()

				return
			}
		}()

		err = container.Invoke(taskResolver)
		if err != nil {
			s.registerTaskError(ctx, holder, err)
			debug.PrintStack()

			return
		}

		logger.Infof("Task terminated with success")

		err = s.store.UpdateTaskStatus(ctx, s.provider, descriptor, models.TaskStatusTerminated, "")
		if err != nil {
			logger.Error("Error updating task status: %s", err)
		}
	}()

	return nil
}

func (s *DefaultTaskScheduler[TaskDescriptor]) stackTask(descriptor TaskDescriptor) error {
	s.logger.WithFields(map[string]interface{}{
		"descriptor": descriptor,
	}).Infof("Stacking task")

	taskDescriptor, err := json.Marshal(descriptor)
	if err != nil {
		return err
	}

	return s.store.UpdateTaskStatus(
		context.Background(), s.provider, taskDescriptor, models.TaskStatusPending, "")
}

var _ Scheduler[struct{}] = &DefaultTaskScheduler[struct{}]{}

func NewDefaultScheduler[TaskDescriptor payments.TaskDescriptor](
	provider models.ConnectorProvider,
	logger sharedlogging.Logger,
	store Repository,
	containerFactory ContainerFactory,
	resolver Resolver[TaskDescriptor],
	maxTasks int,
) *DefaultTaskScheduler[TaskDescriptor] {
	return &DefaultTaskScheduler[TaskDescriptor]{
		provider: provider,
		logger: logger.WithFields(map[string]interface{}{
			"component": "scheduler",
			"provider":  provider,
		}),
		store:            store,
		tasks:            map[string]*taskHolder[TaskDescriptor]{},
		containerFactory: containerFactory,
		maxTasks:         maxTasks,
		resolver:         resolver,
	}
}