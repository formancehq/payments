package integration

import (
	"github.com/formancehq/payments/internal/pkg/payments"
	"github.com/formancehq/payments/internal/pkg/task"
)

type TaskSchedulerFactory[TaskDescriptor payments.TaskDescriptor] interface {
	Make(resolver task.Resolver[TaskDescriptor], maxTasks int) *task.DefaultTaskScheduler[TaskDescriptor]
}

type TaskSchedulerFactoryFn[TaskDescriptor payments.TaskDescriptor] func(resolver task.Resolver[TaskDescriptor],
	maxProcesses int) *task.DefaultTaskScheduler[TaskDescriptor]

func (fn TaskSchedulerFactoryFn[TaskDescriptor]) Make(resolver task.Resolver[TaskDescriptor],
	maxTasks int,
) *task.DefaultTaskScheduler[TaskDescriptor] {
	return fn(resolver, maxTasks)
}
