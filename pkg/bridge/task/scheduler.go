package task

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/numary/go-libs/sharedlogging"
	payments "github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge/ingestion"
	"github.com/numary/payments/pkg/bridge/utils"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/dig"
)

var (
	ErrAlreadyScheduled = errors.New("already scheduled")
	ErrUnableToResolve  = errors.New("unable to resolve task")
)

type IngesterFactory interface {
	Make(ctx context.Context, provider string, task payments.TaskDescriptor) ingestion.Ingester
}
type IngesterFactoryFn func(ctx context.Context, provider string, task payments.TaskDescriptor) ingestion.Ingester

func (fn IngesterFactoryFn) Make(ctx context.Context, provider string, task payments.TaskDescriptor) ingestion.Ingester {
	return fn(ctx, provider, task)
}

var NoOpIngesterFactory IngesterFactoryFn = func(ctx context.Context, provider string, task payments.TaskDescriptor) ingestion.Ingester {
	return ingestion.NoOpIngester()
}

type Resolver[TaskDescriptor payments.TaskDescriptor] interface {
	Resolve(descriptor TaskDescriptor) Task
}
type ResolverFn[TaskDescriptor payments.TaskDescriptor] func(descriptor TaskDescriptor) Task

func (fn ResolverFn[TaskDescriptor]) Resolve(descriptor TaskDescriptor) Task {
	return fn(descriptor)
}

type Scheduler[TaskDescriptor payments.TaskDescriptor] interface {
	Schedule(p TaskDescriptor, restart bool) error
}

type taskHolder[TaskDescriptor payments.TaskDescriptor] struct {
	descriptor TaskDescriptor
	cancel     func()
	logger     sharedlogging.Logger
	stopChan   StopChan
}

type DefaultTaskScheduler[TaskDescriptor payments.TaskDescriptor] struct {
	provider        string
	logger          sharedlogging.Logger
	store           Store[TaskDescriptor]
	ingesterFactory IngesterFactory
	tasks           map[string]*taskHolder[TaskDescriptor]
	idles           utils.FIFO[*taskHolder[TaskDescriptor]]
	mu              sync.Mutex
	maxTasks        int
	resolver        Resolver[TaskDescriptor]
	stopped         bool
}

func (s *DefaultTaskScheduler[TaskDescriptor]) ListTasks(ctx context.Context) ([]payments.TaskState[TaskDescriptor], error) {
	return s.store.ListTaskStates(ctx, s.provider)
}

func (s *DefaultTaskScheduler[TaskDescriptor]) ReadTask(ctx context.Context, descriptor TaskDescriptor) (*payments.TaskState[TaskDescriptor], error) {
	return s.store.ReadTaskState(ctx, s.provider, descriptor)
}

func (s *DefaultTaskScheduler[TaskDescriptor]) Schedule(descriptor TaskDescriptor, restart bool) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	taskId := payments.IDFromDescriptor(descriptor)
	if _, ok := s.tasks[taskId]; ok {
		return ErrAlreadyScheduled
	}

	if !restart {
		_, err := s.store.ReadTaskState(context.Background(), s.provider, descriptor)
		if err == nil {
			return nil
		}
	}

	if len(s.tasks) >= s.maxTasks || s.stopped {
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

	states, err := s.store.ListTaskStatesByStatus(ctx, s.provider, payments.TaskStatusActive)
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
	err := s.store.UpdateTaskStatus(ctx, s.provider, holder.descriptor, payments.TaskStatusFailed, pe)
	if err != nil {
		holder.logger.Error("Error updating task status: %s", pe)
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
	ps, err := s.store.FindTaskAndUpdateStatus(context.Background(), s.provider, descriptor, payments.TaskStatusActive, "")
	if err != nil {
		return errors.Wrap(err, "finding task and update")
	}

	logger := s.logger.WithFields(map[string]interface{}{
		"task-id": payments.IDFromDescriptor(descriptor),
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

	container := dig.New()
	container.Provide(func() context.Context {
		return ctx
	})
	container.Provide(func() Scheduler[TaskDescriptor] {
		return s
	})
	container.Provide(func() ingestion.Ingester {
		return s.ingesterFactory.Make(ctx, s.provider, descriptor)
	})
	container.Provide(func() StopChan {
		s.mu.Lock()
		defer s.mu.Unlock()

		holder.stopChan = make(StopChan, 1)
		return holder.stopChan
	})
	container.Provide(func() sharedlogging.Logger {
		return s.logger
	})
	container.Provide(func() StateResolver {
		return StateResolverFn(func(ctx context.Context, v any) error {
			if ps.State == nil || len(ps.State) == 0 {
				return nil
			}
			return bson.Unmarshal(ps.State, v)
		})
	})

	s.tasks[payments.IDFromDescriptor(descriptor)] = holder
	go func() {
		logger.Infof("Starting task...")

		defer func() {
			defer s.deleteTask(holder)
			if e := recover(); e != nil {
				s.registerTaskError(ctx, holder, e)
				debug.PrintStack()
				return
			}
			logger.Infof("Task terminated with success")
		}()

		err := container.Invoke(task)
		if err != nil {
			s.registerTaskError(ctx, holder, err)
			return
		}

		err = s.store.UpdateTaskStatus(ctx, s.provider, descriptor, payments.TaskStatusTerminated, "")
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
		context.Background(), s.provider, descriptor, payments.TaskStatusPending, "")
}

var _ Scheduler[struct{}] = &DefaultTaskScheduler[struct{}]{}

func NewDefaultScheduler[TaskDescriptor payments.TaskDescriptor](
	provider string,
	logger sharedlogging.Logger,
	store Store[TaskDescriptor],
	ingesterFactory IngesterFactory,
	resolver Resolver[TaskDescriptor],
	maxTasks int,
) *DefaultTaskScheduler[TaskDescriptor] {
	return &DefaultTaskScheduler[TaskDescriptor]{
		provider: provider,
		logger: logger.WithFields(map[string]interface{}{
			"component": "scheduler",
			"provider":  provider,
		}),
		store:           store,
		tasks:           map[string]*taskHolder[TaskDescriptor]{},
		ingesterFactory: ingesterFactory,
		maxTasks:        maxTasks,
		resolver:        resolver,
	}
}
