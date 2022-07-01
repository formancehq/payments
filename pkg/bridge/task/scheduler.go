package task

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/numary/go-libs/sharedlogging"
	payments "github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge/ingestion"
	"github.com/numary/payments/pkg/bridge/utils"
	"github.com/pkg/errors"
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

type Resolver[TaskDescriptor payments.TaskDescriptor, TaskState any] interface {
	Resolve(descriptor TaskDescriptor) Task[TaskDescriptor, TaskState]
}
type ResolverFn[TaskDescriptor payments.TaskDescriptor, TaskState any] func(descriptor TaskDescriptor) Task[TaskDescriptor, TaskState]

func (fn ResolverFn[TaskDescriptor, TaskState]) Resolve(descriptor TaskDescriptor) Task[TaskDescriptor, TaskState] {
	return fn(descriptor)
}

type Scheduler[TaskDescriptor payments.TaskDescriptor] interface {
	Schedule(p TaskDescriptor, restart bool) error
}

type runnerHolder[TaskDescriptor payments.TaskDescriptor, TaskState any] struct {
	descriptor TaskDescriptor
	runner     Task[TaskDescriptor, TaskState]
	logger     sharedlogging.Logger
}

type DefaultTaskScheduler[TaskDescriptor payments.TaskDescriptor, TaskState any] struct {
	provider        string
	logger          sharedlogging.Logger
	store           Store[TaskDescriptor, TaskState]
	ingesterFactory IngesterFactory
	tasks           map[string]*runnerHolder[TaskDescriptor, TaskState]
	idles           utils.FIFO[*runnerHolder[TaskDescriptor, TaskState]]
	mu              sync.Mutex
	maxTasks        int
	resolver        Resolver[TaskDescriptor, TaskState]
	stopped         bool
}

func (s *DefaultTaskScheduler[TaskDescriptor, TaskState]) ListTasks(ctx context.Context) ([]payments.TaskState[TaskDescriptor, TaskState], error) {
	return s.store.ListTaskStates(ctx, s.provider)
}

func (s *DefaultTaskScheduler[TaskDescriptor, TaskState]) ReadTask(ctx context.Context, descriptor TaskDescriptor) (*payments.TaskState[TaskDescriptor, TaskState], error) {
	return s.store.ReadTaskState(ctx, s.provider, descriptor)
}

func (s *DefaultTaskScheduler[TaskDescriptor, TaskState]) Schedule(descriptor TaskDescriptor, restart bool) error {

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

func (s *DefaultTaskScheduler[TaskDescriptor, TaskState]) Shutdown(ctx context.Context) error {

	s.mu.Lock()
	s.stopped = true
	s.mu.Unlock()

	s.logger.Infof("Stopping scheduler...")
	for name, p := range s.tasks {
		p.logger.Debugf("Stopping task")
		err := p.runner.Cancel(ctx)
		if err != nil {
			return err
		}
		delete(s.tasks, name)
	}
	return nil
}

func (s *DefaultTaskScheduler[TaskDescriptor, TaskState]) Restore(ctx context.Context) error {

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

func (s *DefaultTaskScheduler[TaskDescriptor, TaskState]) registerTaskError(ctx context.Context, holder *runnerHolder[TaskDescriptor, TaskState], taskErr any) {

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

func (s *DefaultTaskScheduler[TaskDescriptor, TaskState]) deleteTask(holder *runnerHolder[TaskDescriptor, TaskState]) {
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

func (s *DefaultTaskScheduler[TaskDescriptor, TaskState]) startTask(descriptor TaskDescriptor) error {
	ps, err := s.store.FindTaskAndUpdateStatus(context.Background(), s.provider, descriptor, payments.TaskStatusActive, "")
	if err != nil {
		return errors.Wrap(err, "finding task and update")
	}

	logger := s.logger.WithFields(map[string]interface{}{
		"task-id": payments.IDFromDescriptor(descriptor),
	})

	runner := s.resolver.Resolve(descriptor)
	if runner == nil {
		return ErrUnableToResolve
	}
	holder := &runnerHolder[TaskDescriptor, TaskState]{
		runner:     runner,
		logger:     logger,
		descriptor: descriptor,
	}
	s.tasks[payments.IDFromDescriptor(descriptor)] = holder
	ctx := context.Background()
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
		err := runner.Run(&taskContextImpl[TaskDescriptor, TaskState]{
			provider:  s.provider,
			scheduler: s,
			logger:    logger,
			ctx:       ctx,
			ingester:  s.ingesterFactory.Make(ctx, s.provider, descriptor),
			state:     ps.State,
		})
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

func (s *DefaultTaskScheduler[TaskDescriptor, TaskState]) stackTask(descriptor TaskDescriptor) error {
	s.logger.WithFields(map[string]interface{}{
		"descriptor": descriptor,
	}).Infof("Stacking task")
	return s.store.UpdateTaskStatus(
		context.Background(), s.provider, descriptor, payments.TaskStatusPending, "")
}

var _ Scheduler[struct{}] = &DefaultTaskScheduler[struct{}, struct{}]{}

func NewDefaultScheduler[TaskDescriptor payments.TaskDescriptor, TaskState any](
	provider string,
	logger sharedlogging.Logger,
	store Store[TaskDescriptor, TaskState],
	ingesterFactory IngesterFactory,
	resolver Resolver[TaskDescriptor, TaskState],
	maxTasks int,
) *DefaultTaskScheduler[TaskDescriptor, TaskState] {
	return &DefaultTaskScheduler[TaskDescriptor, TaskState]{
		provider: provider,
		logger: logger.WithFields(map[string]interface{}{
			"component": "scheduler",
			"provider":  provider,
		}),
		store:           store,
		tasks:           map[string]*runnerHolder[TaskDescriptor, TaskState]{},
		ingesterFactory: ingesterFactory,
		maxTasks:        maxTasks,
		resolver:        resolver,
	}
}
