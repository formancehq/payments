package dummypay

import (
	"fmt"

	"github.com/numary/payments/pkg/bridge/task"
)

// taskKey defines a unique key of the task.
type taskKey string

// TaskDescriptor represents a task descriptor.
type TaskDescriptor struct {
	Key      taskKey
	FileName string
}

// handleResolve resolves a task execution request based on the task descriptor.
func handleResolve(config Config, descriptor TaskDescriptor) task.Task {
	switch descriptor.Key {
	case taskKeyReadFiles:
		return taskReadFiles(config)
	case taskKeyIngest:
		return taskIngest(config, descriptor)
	case taskKeyGenerateFiles:
		return taskGenerateFiles(config)
	}

	// This should never happen.
	return func() error {
		return fmt.Errorf("key '%s': %w", descriptor.Key, ErrMissingTask)
	}
}
