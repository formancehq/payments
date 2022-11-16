package dummypay

import (
	"fmt"

	"github.com/numary/payments/internal/pkg/task"
)

// taskKey defines a unique key of the task.
type taskKey string

// TaskDescriptor represents a task descriptor.
type TaskDescriptor struct {
	Key      taskKey
	FileName string
}

// handleResolve resolves a task execution request based on the task descriptor.
func handleResolve(config Config, descriptor TaskDescriptor, fs fs) task.Task {
	switch descriptor.Key {
	case taskKeyReadFiles:
		return taskReadFiles(config, fs)
	case taskKeyIngest:
		return taskIngest(config, descriptor, fs)
	case taskKeyGenerateFiles:
		return taskGenerateFiles(config, fs)
	}

	// This should never happen.
	return func() error {
		return fmt.Errorf("key '%s': %w", descriptor.Key, ErrMissingTask)
	}
}
