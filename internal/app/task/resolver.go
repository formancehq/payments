package task

import (
	"github.com/formancehq/payments/internal/app/payments"
)

type Resolver[TaskDescriptor payments.TaskDescriptor] interface {
	Resolve(descriptor TaskDescriptor) Task
}

type ResolverFn[TaskDescriptor payments.TaskDescriptor] func(descriptor TaskDescriptor) Task

func (fn ResolverFn[TaskDescriptor]) Resolve(descriptor TaskDescriptor) Task {
	return fn(descriptor)
}
