package dummypay

import (
	"context"
	"reflect"
	"testing"

	"github.com/numary/payments/internal/pkg/payments"
	task3 "github.com/numary/payments/internal/pkg/task"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/stretchr/testify/assert"
)

// Create a minimal mock for connector installation.
type (
	mockConnectorContext[TaskDescriptor payments.TaskDescriptor] struct {
		ctx context.Context
	}
	mockScheduler[TaskDescriptor payments.TaskDescriptor] struct{}
)

func (mcc *mockConnectorContext[TaskDescriptor]) Context() context.Context {
	return mcc.ctx
}

func (mcc mockScheduler[TaskDescriptor]) Schedule(p TaskDescriptor, restart bool) error {
	return nil
}

func (mcc *mockConnectorContext[TaskDescriptor]) Scheduler() task3.Scheduler[TaskDescriptor] {
	return mockScheduler[TaskDescriptor]{}
}

func TestConnector(t *testing.T) {
	t.Parallel()

	config := Config{}
	logger := sharedlogging.GetLogger(context.Background())

	fileSystem := newTestFS()

	connector := newConnector(logger, config, fileSystem)

	err := connector.Install(new(mockConnectorContext[TaskDescriptor]))
	assert.NoErrorf(t, err, "Install() failed")

	testCases := []struct {
		key  taskKey
		task task3.Task
	}{
		{taskKeyReadFiles, taskReadFiles(config, fileSystem)},
		{taskKeyGenerateFiles, taskGenerateFiles(config, fileSystem)},
		{taskKeyIngest, taskIngest(config, TaskDescriptor{}, fileSystem)},
	}

	for _, testCase := range testCases {
		assert.EqualValues(t,
			reflect.ValueOf(testCase.task).String(),
			reflect.ValueOf(connector.Resolve(TaskDescriptor{Key: testCase.key})).String(),
		)
	}

	assert.EqualValues(t,
		reflect.ValueOf(func() error { return nil }).String(),
		reflect.ValueOf(connector.Resolve(TaskDescriptor{Key: "test"})).String(),
	)

	assert.NoError(t, connector.Uninstall(context.Background()))
}
