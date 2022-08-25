package task

import (
	"context"

	"github.com/numary/payments/pkg/core"
)

type ConnectorContext[TaskDescriptor core.TaskDescriptor] interface {
	Context() context.Context
	Scheduler() Scheduler[TaskDescriptor]
}

type connectorContext[TaskDescriptor core.TaskDescriptor] struct {
	ctx       context.Context
	scheduler Scheduler[TaskDescriptor]
}

func (ctx *connectorContext[TaskDescriptor]) Context() context.Context {
	return ctx.ctx
}
func (ctx *connectorContext[TaskDescriptor]) Scheduler() Scheduler[TaskDescriptor] {
	return ctx.scheduler
}

func NewConnectorContext[TaskDescriptor core.TaskDescriptor](ctx context.Context, scheduler Scheduler[TaskDescriptor]) *connectorContext[TaskDescriptor] {
	return &connectorContext[TaskDescriptor]{
		ctx:       ctx,
		scheduler: scheduler,
	}
}
