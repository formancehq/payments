package integration

import (
	"context"

	"github.com/formancehq/payments/internal/app/payments"
	"github.com/formancehq/payments/internal/app/task"
)

// Connector provide entry point to a payment provider.
type Connector[TaskDescriptor payments.TaskDescriptor] interface {
	// Install is used to start the connector. The implementation if in charge of scheduling all required resources.
	Install(ctx task.ConnectorContext[TaskDescriptor]) error
	// Uninstall is used to uninstall the connector. It has to close all related resources opened by the connector.
	Uninstall(ctx context.Context) error
	// Resolve is used to recover state of a failed or restarted task
	Resolve(descriptor TaskDescriptor) task.Task
}

type ConnectorBuilder[TaskDescriptor payments.TaskDescriptor] struct {
	name      string
	uninstall func(ctx context.Context) error
	resolve   func(descriptor TaskDescriptor) task.Task
	install   func(ctx task.ConnectorContext[TaskDescriptor]) error
}

func (b *ConnectorBuilder[TaskDescriptor]) WithUninstall(
	uninstallFunction func(ctx context.Context) error,
) *ConnectorBuilder[TaskDescriptor] {
	b.uninstall = uninstallFunction

	return b
}

func (b *ConnectorBuilder[TaskDescriptor]) WithResolve(
	resolveFunction func(name TaskDescriptor) task.Task,
) *ConnectorBuilder[TaskDescriptor] {
	b.resolve = resolveFunction

	return b
}

func (b *ConnectorBuilder[TaskDescriptor]) WithInstall(
	installFunction func(ctx task.ConnectorContext[TaskDescriptor]) error,
) *ConnectorBuilder[TaskDescriptor] {
	b.install = installFunction

	return b
}

func (b *ConnectorBuilder[TaskDescriptor]) Build() Connector[TaskDescriptor] {
	return &BuiltConnector[TaskDescriptor]{
		name:      b.name,
		uninstall: b.uninstall,
		resolve:   b.resolve,
		install:   b.install,
	}
}

func NewConnectorBuilder[TaskDescriptor payments.TaskDescriptor]() *ConnectorBuilder[TaskDescriptor] {
	return &ConnectorBuilder[TaskDescriptor]{}
}

type BuiltConnector[TaskDescriptor payments.TaskDescriptor] struct {
	name      string
	uninstall func(ctx context.Context) error
	resolve   func(name TaskDescriptor) task.Task
	install   func(ctx task.ConnectorContext[TaskDescriptor]) error
}

func (b *BuiltConnector[TaskDescriptor]) Name() string {
	return b.name
}

func (b *BuiltConnector[TaskDescriptor]) Install(ctx task.ConnectorContext[TaskDescriptor]) error {
	if b.install != nil {
		return b.install(ctx)
	}

	return nil
}

func (b *BuiltConnector[TaskDescriptor]) Uninstall(ctx context.Context) error {
	if b.uninstall != nil {
		return b.uninstall(ctx)
	}

	return nil
}

func (b *BuiltConnector[TaskDescriptor]) Resolve(name TaskDescriptor) task.Task {
	if b.resolve != nil {
		return b.resolve(name)
	}

	return nil
}

var _ Connector[struct{}] = &BuiltConnector[struct{}]{}
