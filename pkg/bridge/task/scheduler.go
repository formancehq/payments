package task

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/core"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/dig"
)

var (
	ErrAlreadyScheduled = errors.New("already scheduled")
	ErrUnableToResolve  = errors.New("unable to resolve task")
)

type Resolver[TaskDescriptor core.TaskDescriptor] interface {
	Resolve(descriptor TaskDescriptor) Task
}
type ResolverFn[TaskDescriptor core.TaskDescriptor] func(descriptor TaskDescriptor) Task

func (fn ResolverFn[TaskDescriptor]) Resolve(descriptor TaskDescriptor) Task {
	return fn(descriptor)
}

type ContainerFactory interface {
	Create(ctx context.Context, descriptor core.TaskDescriptor) (*dig.Container, error)
}
type ContainerFactoryFn func(ctx context.Context, descriptor core.TaskDescriptor) (*dig.Container, error)

func (fn ContainerFactoryFn) Create(ctx context.Context, descriptor core.TaskDescriptor) (*dig.Container, error) {
	return fn(ctx, descriptor)
}

var DefaultContainerFactory = ContainerFactoryFn(func(ctx context.Context, descriptor core.TaskDescriptor) (*dig.Container, error) {
	return dig.New(), nil
})

type Scheduler[TaskDescriptor core.TaskDescriptor] interface {
	Schedule(p TaskDescriptor, restart bool) error
}

type taskHolder[TaskDescriptor core.TaskDescriptor] struct {
	descriptor TaskDescriptor
	cancel     func()
	logger     sharedlogging.Logger
	stopChan   StopChan
}

type DefaultTaskScheduler[TaskDescriptor core.TaskDescriptor] struct {
	provider         string
	logger           sharedlogging.Logger
	store            Store[TaskDescriptor]
	containerFactory ContainerFactory
	tasks            map[string]*taskHolder[TaskDescriptor]
	mu               sync.Mutex
	maxTasks         int
	resolver         Resolver[TaskDescriptor]
	stopped          bool
}

func (s *DefaultTaskScheduler[TaskDescriptor]) ListTasks(ctx context.Context) ([]core.TaskState[TaskDescriptor], error) {
	return s.store.ListTaskStates(ctx, s.provider)
}

func (s *DefaultTaskScheduler[TaskDescriptor]) ReadTask(ctx context.Context, descriptor TaskDescriptor) (*core.TaskState[TaskDescriptor], error) {
	return s.store.ReadTaskState(ctx, s.provider, descriptor)
}

func (s *DefaultTaskScheduler[TaskDescriptor]) Schedule(descriptor TaskDescriptor, restart bool) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	taskId := core.IDFromDescriptor(descriptor)
	if _, ok := s.tasks[taskId]; ok {
		return ErrAlreadyScheduled
	}

	if !restart {
		_, err := s.store.ReadTaskState(context.Background(), s.provider, descriptor)
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

	err := s.startTask(descriptor)
	if err != nil {
		return errors.Wrap(err, "starting task")
	}

	return nil
}

func (s *DefaultTaskScheduler[TaskDescriptor]) Shutdown(ctx context.Context) error {

	s.mu.Lock()
	s.stopped = true
	s.mu.Unlock()

	s.logger.Infof("Stopping scheduler...")
	for name, p := range s.tasks {
		p.logger.Debugf("Stopping task")
		if p.stopChan != nil {
			errCh := make(chan struct{})
			p.stopChan <- errCh
			select {
			case <-errCh:
			case <-time.After(time.Second): // TODO: Make configurable
				p.logger.Debugf("Stopping using stop chan timeout, cancelling context")
				p.cancel()
			}
		} else {
			p.cancel()
		}
		delete(s.tasks, name)
	}
	return nil
}

func (s *DefaultTaskScheduler[TaskDescriptor]) Restore(ctx context.Context) error {

	states, err := s.store.ListTaskStatesByStatus(ctx, s.provider, core.TaskStatusActive)
	if err != nil {
		return err
	}

	for _, state := range states {
		err = s.startTask(state.Descriptor)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *DefaultTaskScheduler[TaskDescriptor]) registerTaskError(ctx context.Context, holder *taskHolder[TaskDescriptor], taskErr any) {

	var pe string
	switch v := taskErr.(type) {
	case error:
		pe = v.Error()
	default:
		pe = fmt.Sprintf("%s", v)
	}

	holder.logger.Errorf("Task terminated with error: %s", taskErr)
	err := s.store.UpdateTaskStatus(ctx, s.provider, holder.descriptor, core.TaskStatusFailed, pe)
	if err != nil {
		holder.logger.Error("Error updating task status: %s", pe)
	}
}

func (s *DefaultTaskScheduler[TaskDescriptor]) deleteTask(holder *taskHolder[TaskDescriptor]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tasks, core.IDFromDescriptor(holder.descriptor))

	if s.stopped {
		return
	}

	oldestPendingTask, err := s.store.ReadOldestPendingTask(context.Background(), s.provider)
	if err != nil {
		if err == ErrNotFound {
			return
		}
		sharedlogging.Error(err)
		return
	}

	p := s.resolver.Resolve(oldestPendingTask.Descriptor)
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

func (s *DefaultTaskScheduler[TaskDescriptor]) startTask(descriptor TaskDescriptor) error {
	ps, err := s.store.FindTaskAndUpdateStatus(context.Background(), s.provider, descriptor, core.TaskStatusActive, "")
	if err != nil {
		return errors.Wrap(err, "finding task and update")
	}

	taskId := core.IDFromDescriptor(descriptor)
	logger := s.logger.WithFields(map[string]interface{}{
		"task-id": taskId,
	})

	task := s.resolver.Resolve(descriptor)
	if task == nil {
		return ErrUnableToResolve
	}

	//TODO: Check task using reflection

	ctx, cancel := context.WithCancel(context.Background())
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
			if ps.State == nil || len(ps.State) == 0 {
				return nil
			}
			return bson.Unmarshal(ps.State, v)
		})
	})
	if err != nil {
		panic(err)
	}

	s.tasks[core.IDFromDescriptor(descriptor)] = holder
	go func() {
		logger.Infof("Starting task...")

		defer func() {
			defer s.deleteTask(holder)
			if e := recover(); e != nil {
				s.registerTaskError(ctx, holder, e)
				debug.PrintStack()
				return
			}
		}()

		err := container.Invoke(task)
		if err != nil {
			s.registerTaskError(ctx, holder, err)
			return
		}
		logger.Infof("Task terminated with success")

		err = s.store.UpdateTaskStatus(ctx, s.provider, descriptor, core.TaskStatusTerminated, "")
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
	return s.store.UpdateTaskStatus(
		context.Background(), s.provider, descriptor, core.TaskStatusPending, "")
}

var _ Scheduler[struct{}] = &DefaultTaskScheduler[struct{}]{}

func NewDefaultScheduler[TaskDescriptor core.TaskDescriptor](
	provider string,
	logger sharedlogging.Logger,
	store Store[TaskDescriptor],
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
