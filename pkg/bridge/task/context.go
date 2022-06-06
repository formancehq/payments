package task

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge/ingestion"
)

type Context[TaskDescriptor payments.TaskDescriptor, TaskState any] interface {
	Context() context.Context
	Scheduler() Scheduler[TaskDescriptor]
	Logger() sharedlogging.Logger
	Ingester() ingestion.Ingester
	State() TaskState
	WithContext(ctx context.Context) Context[TaskDescriptor, TaskState]
}

type ConnectorContext[TaskDescriptor payments.TaskDescriptor] interface {
	Context() context.Context
	Scheduler() Scheduler[TaskDescriptor]
}

type connectorContext[TaskDescriptor payments.TaskDescriptor] struct {
	ctx       context.Context
	scheduler Scheduler[TaskDescriptor]
}

func (ctx *connectorContext[TaskDescriptor]) Context() context.Context {
	return ctx.ctx
}
func (ctx *connectorContext[TaskDescriptor]) Scheduler() Scheduler[TaskDescriptor] {
	return ctx.scheduler
}

func NewConnectorContext[TaskDescriptor payments.TaskDescriptor](ctx context.Context, scheduler Scheduler[TaskDescriptor]) *connectorContext[TaskDescriptor] {
	return &connectorContext[TaskDescriptor]{
		ctx:       ctx,
		scheduler: scheduler,
	}
}
