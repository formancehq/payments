package integration

import (
	"context"
	"github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge/task"
)

// Connector provide entry point to a payment provider
// It requires a payments.ConnectorConfigObject representing the configuration of the specific payment provider
// as well as a payments.ConnectorState object which represents the state of the connector
type Connector[TaskDescriptor payments.TaskDescriptor, TaskState any] interface {
	// Install is used to start the connector. The implementation if in charge of scheduling all required resources.
	Install(ctx task.ConnectorContext[TaskDescriptor]) error
	// Uninstall is used to uninstall the connector. It has to close all related resources opened by the connector.
	Uninstall(ctx context.Context) error
	// Resolve is used to recover state of a failed or restarted task
	Resolve(descriptor TaskDescriptor) task.Task[TaskDescriptor, TaskState]
}

type ConnectorBuilder[TaskDescriptor payments.TaskDescriptor, TaskState any] struct {
	name      string
	uninstall func(ctx context.Context) error
	resolve   func(descriptor TaskDescriptor) task.Task[TaskDescriptor, TaskState]
	install   func(ctx task.ConnectorContext[TaskDescriptor]) error
}

func (b *ConnectorBuilder[TaskDescriptor, TaskState]) WithUninstall(uninstallFunction func(ctx context.Context) error) *ConnectorBuilder[TaskDescriptor, TaskState] {
	b.uninstall = uninstallFunction
	return b
}

func (b *ConnectorBuilder[TaskDescriptor, TaskState]) WithResolve(resolveFunction func(name TaskDescriptor) task.Task[TaskDescriptor, TaskState]) *ConnectorBuilder[TaskDescriptor, TaskState] {
	b.resolve = resolveFunction
	return b
}

func (b *ConnectorBuilder[TaskDescriptor, TaskState]) WithInstall(installFunction func(ctx task.ConnectorContext[TaskDescriptor]) error) *ConnectorBuilder[TaskDescriptor, TaskState] {
	b.install = installFunction
	return b
}

func (b *ConnectorBuilder[TaskDescriptor, TaskState]) Build() Connector[TaskDescriptor, TaskState] {
	return &BuiltConnector[TaskDescriptor, TaskState]{
		name:      b.name,
		uninstall: b.uninstall,
		resolve:   b.resolve,
		install:   b.install,
	}
}

func NewConnectorBuilder[TaskDescriptor payments.TaskDescriptor, TaskState any]() *ConnectorBuilder[TaskDescriptor, TaskState] {
	return &ConnectorBuilder[TaskDescriptor, TaskState]{}
}

type BuiltConnector[TaskDescriptor payments.TaskDescriptor, TaskState any] struct {
	name      string
	uninstall func(ctx context.Context) error
	resolve   func(name TaskDescriptor) task.Task[TaskDescriptor, TaskState]
	install   func(ctx task.ConnectorContext[TaskDescriptor]) error
}

func (b *BuiltConnector[TaskDescriptor, TaskState]) Name() string {
	return b.name
}

func (b *BuiltConnector[TaskDescriptor, TaskState]) Install(ctx task.ConnectorContext[TaskDescriptor]) error {
	if b.install != nil {
		return b.install(ctx)
	}
	return nil
}

func (b *BuiltConnector[TaskDescriptor, TaskState]) Uninstall(ctx context.Context) error {
	if b.uninstall != nil {
		return b.uninstall(ctx)
	}
	return nil
}

func (b *BuiltConnector[TaskDescriptor, TaskState]) Resolve(name TaskDescriptor) task.Task[TaskDescriptor, TaskState] {
	if b.resolve != nil {
		return b.resolve(name)
	}
	return nil
}

var _ Connector[struct{}, struct{}] = &BuiltConnector[struct{}, struct{}]{}
