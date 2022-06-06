package integration

import (
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg"
)

type Loader[ConnectorConfig payments.ConnectorConfigObject, TaskDescriptor payments.TaskDescriptor, TaskState any] interface {
	Name() string
	Load(logger sharedlogging.Logger, config ConnectorConfig) Connector[TaskDescriptor, TaskState]
	// ApplyDefaults is used to fill default values of the provided configuration object
	ApplyDefaults(t ConnectorConfig) ConnectorConfig
	AllowTasks() int
}

type LoaderBuilder[ConnectorConfig payments.ConnectorConfigObject, TaskDescriptor payments.TaskDescriptor, TaskState any] struct {
	loadFunction  func(logger sharedlogging.Logger, config ConnectorConfig) Connector[TaskDescriptor, TaskState]
	applyDefaults func(t ConnectorConfig) ConnectorConfig
	name          string
	allowedTasks  int
}

func (b *LoaderBuilder[ConnectorConfig, TaskDescriptor, TaskState]) WithLoad(loadFunction func(logger sharedlogging.Logger, config ConnectorConfig) Connector[TaskDescriptor, TaskState]) *LoaderBuilder[ConnectorConfig, TaskDescriptor, TaskState] {
	b.loadFunction = loadFunction
	return b
}

func (b *LoaderBuilder[ConnectorConfig, TaskDescriptor, TaskState]) WithApplyDefaults(applyDefaults func(t ConnectorConfig) ConnectorConfig) *LoaderBuilder[ConnectorConfig, TaskDescriptor, TaskState] {
	b.applyDefaults = applyDefaults
	return b
}

func (b *LoaderBuilder[ConnectorConfig, TaskDescriptor, TaskState]) WithAllowedTasks(v int) *LoaderBuilder[ConnectorConfig, TaskDescriptor, TaskState] {
	b.allowedTasks = v
	return b
}

func (b *LoaderBuilder[ConnectorConfig, TaskDescriptor, TaskState]) Build() *BuiltLoader[ConnectorConfig, TaskDescriptor, TaskState] {
	return &BuiltLoader[ConnectorConfig, TaskDescriptor, TaskState]{
		loadFunction:  b.loadFunction,
		applyDefaults: b.applyDefaults,
		name:          b.name,
		allowedTasks:  b.allowedTasks,
	}
}

func NewLoaderBuilder[ConnectorConfig payments.ConnectorConfigObject, TaskDescriptor payments.TaskDescriptor, TaskState any](name string) *LoaderBuilder[ConnectorConfig, TaskDescriptor, TaskState] {
	return &LoaderBuilder[ConnectorConfig, TaskDescriptor, TaskState]{
		name: name,
	}
}

type BuiltLoader[ConnectorConfig payments.ConnectorConfigObject, TaskDescriptor payments.TaskDescriptor, TaskState any] struct {
	loadFunction  func(logger sharedlogging.Logger, config ConnectorConfig) Connector[TaskDescriptor, TaskState]
	applyDefaults func(t ConnectorConfig) ConnectorConfig
	name          string
	allowedTasks  int
}

func (b *BuiltLoader[ConnectorConfig, TaskDescriptor, TaskState]) AllowTasks() int {
	return b.allowedTasks
}

func (b *BuiltLoader[ConnectorConfig, TaskDescriptor, TaskState]) Name() string {
	return b.name
}

func (b *BuiltLoader[ConnectorConfig, TaskDescriptor, TaskState]) Load(logger sharedlogging.Logger, config ConnectorConfig) Connector[TaskDescriptor, TaskState] {
	return b.loadFunction(logger, config)
}

func (b *BuiltLoader[ConnectorConfig, TaskDescriptor, TaskState]) ApplyDefaults(t ConnectorConfig) ConnectorConfig {
	if b.applyDefaults != nil {
		return b.applyDefaults(t)
	}
	return t
}

var _ Loader[payments.EmptyConnectorConfig, struct{}, struct{}] = &BuiltLoader[payments.EmptyConnectorConfig, struct{}, struct{}]{}
