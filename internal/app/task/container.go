package task

import (
	"context"

	"github.com/formancehq/payments/internal/app/payments"

	"go.uber.org/dig"
)

type ContainerFactory interface {
	Create(ctx context.Context, descriptor payments.TaskDescriptor) (*dig.Container, error)
}
type ContainerFactoryFn func(ctx context.Context, descriptor payments.TaskDescriptor) (*dig.Container, error)

func (fn ContainerFactoryFn) Create(ctx context.Context, descriptor payments.TaskDescriptor) (*dig.Container, error) {
	return fn(ctx, descriptor)
}

// nolint: gochecknoglobals,golint,stylecheck // allow global
var DefaultContainerFactory = ContainerFactoryFn(func(ctx context.Context,
	descriptor payments.TaskDescriptor,
) (*dig.Container, error) {
	return dig.New(), nil
})
