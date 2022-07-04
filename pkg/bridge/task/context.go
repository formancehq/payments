package task

import (
	"context"

	"github.com/numary/payments/pkg"
)

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
