package task

import (
	"context"

	"github.com/formancehq/payments/internal/app/payments"
)

type ConnectorContext[TaskDescriptor payments.TaskDescriptor] interface {
	Context() context.Context
	Scheduler() Scheduler[TaskDescriptor]
}

type ConnectorCtx[TaskDescriptor payments.TaskDescriptor] struct {
	ctx       context.Context
	scheduler Scheduler[TaskDescriptor]
}

func (ctx *ConnectorCtx[TaskDescriptor]) Context() context.Context {
	return ctx.ctx
}

func (ctx *ConnectorCtx[TaskDescriptor]) Scheduler() Scheduler[TaskDescriptor] {
	return ctx.scheduler
}

func NewConnectorContext[TaskDescriptor payments.TaskDescriptor](ctx context.Context,
	scheduler Scheduler[TaskDescriptor],
) *ConnectorCtx[TaskDescriptor] {
	return &ConnectorCtx[TaskDescriptor]{
		ctx:       ctx,
		scheduler: scheduler,
	}
}
