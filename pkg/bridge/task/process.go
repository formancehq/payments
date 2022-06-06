package task

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge/ingestion"
)

type Runner[TaskDescriptor payments.TaskDescriptor, TaskState any] interface {
	Run(ctx Context[TaskDescriptor, TaskState]) error
}
type RunnerFn[TaskDescriptor payments.TaskDescriptor, TaskState any] func(ctx Context[TaskDescriptor, TaskState]) error

func (fn RunnerFn[TaskDescriptor, TaskState]) Run(ctx Context[TaskDescriptor, TaskState]) error {
	return fn(ctx)
}

var _ Runner[any, any] = RunnerFn[any, any](func(Context[any, any]) error { return nil })

type Task[Descriptor payments.TaskDescriptor, State any] interface {
	Runner[Descriptor, State]
	Cancel(ctx context.Context) error
}

type functionTask[Descriptor payments.TaskDescriptor, State any] struct {
	runner     Runner[Descriptor, State]
	cancel     func()
	terminated chan struct{}
}

func (f *functionTask[TaskDescriptor, TaskState]) Run(ctx Context[TaskDescriptor, TaskState]) error {
	subContext, cancel := context.WithCancel(ctx.Context())
	f.cancel, f.terminated = cancel, make(chan struct{})
	defer close(f.terminated)
	return f.runner.Run(ctx.WithContext(subContext))
}

func (f *functionTask[TaskDescriptor, TaskState]) Cancel(ctx context.Context) error {
	if f.cancel != nil {
		f.cancel()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-f.terminated:
			return nil
		}
	}
	return nil
}

var _ Task[struct{}, struct{}] = &functionTask[struct{}, struct{}]{}

func NewFunctionTask[TaskDescriptor payments.TaskDescriptor, TaskState any](runner Runner[TaskDescriptor, TaskState]) *functionTask[TaskDescriptor, TaskState] {
	return &functionTask[TaskDescriptor, TaskState]{
		runner: runner,
	}
}

type taskContextImpl[TaskDescriptor payments.TaskDescriptor, TaskState any] struct {
	provider  string
	scheduler Scheduler[TaskDescriptor]
	logger    sharedlogging.Logger
	ctx       context.Context
	ingester  ingestion.Ingester
	state     TaskState
}

func (ctx *taskContextImpl[TaskDescriptor, TaskState]) Context() context.Context {
	return ctx.ctx
}

func (ctx *taskContextImpl[TaskDescriptor, TaskState]) Scheduler() Scheduler[TaskDescriptor] {
	return ctx.scheduler
}

func (ctx *taskContextImpl[TaskDescriptor, TaskState]) Logger() sharedlogging.Logger {
	return ctx.logger
}

func (ctx *taskContextImpl[TaskDescriptor, TaskState]) Ingester() ingestion.Ingester {
	return ctx.ingester
}

func (ctx *taskContextImpl[TaskDescriptor, TaskState]) State() TaskState {
	return ctx.state
}

func (ctx *taskContextImpl[TaskDescriptor, TaskState]) WithContext(c context.Context) Context[TaskDescriptor, TaskState] {
	ctx.ctx = c
	return ctx
}

var _ Context[struct{}, struct{}] = &taskContextImpl[struct{}, struct{}]{}
