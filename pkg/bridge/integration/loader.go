package integration

import (
	"github.com/numary/go-libs/sharedlogging"
	payments "github.com/numary/payments/pkg"
)

type Loader[ConnectorConfig payments.ConnectorConfigObject, TaskDescriptor payments.TaskDescriptor] interface {
	Name() string
	Load(logger sharedlogging.Logger, config ConnectorConfig) Connector[TaskDescriptor]
	// ApplyDefaults is used to fill default values of the provided configuration object
	ApplyDefaults(t ConnectorConfig) ConnectorConfig
	// AllowTasks define how many task the connector can run
	// If too many tasks are scheduled by the connector,
	// those will be set to pending state and restarted later when some other tasks will be terminated
	AllowTasks() int
}

type LoaderBuilder[ConnectorConfig payments.ConnectorConfigObject, TaskDescriptor payments.TaskDescriptor] struct {
	loadFunction  func(logger sharedlogging.Logger, config ConnectorConfig) Connector[TaskDescriptor]
	applyDefaults func(t ConnectorConfig) ConnectorConfig
	name          string
	allowedTasks  int
}

func (b *LoaderBuilder[ConnectorConfig, TaskDescriptor]) WithLoad(loadFunction func(logger sharedlogging.Logger, config ConnectorConfig) Connector[TaskDescriptor]) *LoaderBuilder[ConnectorConfig, TaskDescriptor] {
	b.loadFunction = loadFunction
	return b
}

func (b *LoaderBuilder[ConnectorConfig, TaskDescriptor]) WithApplyDefaults(applyDefaults func(t ConnectorConfig) ConnectorConfig) *LoaderBuilder[ConnectorConfig, TaskDescriptor] {
	b.applyDefaults = applyDefaults
	return b
}

func (b *LoaderBuilder[ConnectorConfig, TaskDescriptor]) WithAllowedTasks(v int) *LoaderBuilder[ConnectorConfig, TaskDescriptor] {
	b.allowedTasks = v
	return b
}

func (b *LoaderBuilder[ConnectorConfig, TaskDescriptor]) Build() *BuiltLoader[ConnectorConfig, TaskDescriptor] {
	return &BuiltLoader[ConnectorConfig, TaskDescriptor]{
		loadFunction:  b.loadFunction,
		applyDefaults: b.applyDefaults,
		name:          b.name,
		allowedTasks:  b.allowedTasks,
	}
}

func NewLoaderBuilder[ConnectorConfig payments.ConnectorConfigObject, TaskDescriptor payments.TaskDescriptor](name string) *LoaderBuilder[ConnectorConfig, TaskDescriptor] {
	return &LoaderBuilder[ConnectorConfig, TaskDescriptor]{
		name: name,
	}
}

type BuiltLoader[ConnectorConfig payments.ConnectorConfigObject, TaskDescriptor payments.TaskDescriptor] struct {
	loadFunction  func(logger sharedlogging.Logger, config ConnectorConfig) Connector[TaskDescriptor]
	applyDefaults func(t ConnectorConfig) ConnectorConfig
	name          string
	allowedTasks  int
}

func (b *BuiltLoader[ConnectorConfig, TaskDescriptor]) AllowTasks() int {
	return b.allowedTasks
}

func (b *BuiltLoader[ConnectorConfig, TaskDescriptor]) Name() string {
	return b.name
}

func (b *BuiltLoader[ConnectorConfig, TaskDescriptor]) Load(logger sharedlogging.Logger, config ConnectorConfig) Connector[TaskDescriptor] {
	if b.loadFunction != nil {
		return b.loadFunction(logger, config)
	}
	return b.loadFunction(logger, config)
}

func (b *BuiltLoader[ConnectorConfig, TaskDescriptor]) ApplyDefaults(t ConnectorConfig) ConnectorConfig {
	if b.applyDefaults != nil {
		return b.applyDefaults(t)
	}
	return t
}

var _ Loader[payments.EmptyConnectorConfig, struct{}] = &BuiltLoader[payments.EmptyConnectorConfig, struct{}]{}
