package activities

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/mock/gomock"
)

func TestCreateOutboxPublisherSchedule(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("success - create schedule", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Setup mocks
		mockScheduleClient := NewMockScheduleClient(ctrl)
		mockHandle := NewMockScheduleHandle(ctrl)
		mockClient := NewMockClient(ctrl)

		// Expectations
		mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()
		mockScheduleClient.EXPECT().GetHandle(ctx, "test-stack-outbox-publisher").Return(mockHandle)
		mockHandle.EXPECT().Describe(ctx).Return(nil, serviceerror.NewNotFound("not found"))
		mockScheduleClient.EXPECT().Create(ctx, gomock.Any()).Return(mockHandle, nil)

		activity := &Activities{
			temporalClient: mockClient,
		}

		// Execute activity
		err := activity.CreateOutboxPublisherSchedule(ctx, "test-stack-outbox-publisher", "test-stack-default", "test-stack")

		// Verify
		require.NoError(t, err)
	})

	t.Run("error - create schedule fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Setup mocks
		mockScheduleClient := NewMockScheduleClient(ctrl)
		mockHandle := NewMockScheduleHandle(ctrl)
		mockClient := NewMockClient(ctrl)

		// Expectations
		mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()
		mockScheduleClient.EXPECT().GetHandle(ctx, "test-stack-outbox-publisher").Return(mockHandle)
		mockHandle.EXPECT().Describe(ctx).Return(nil, serviceerror.NewNotFound("not found"))
		mockScheduleClient.EXPECT().Create(ctx, gomock.Any()).Return(nil, fmt.Errorf("create error"))

		activity := &Activities{
			temporalClient: mockClient,
		}

		// Execute activity
		err := activity.CreateOutboxPublisherSchedule(ctx, "test-stack-outbox-publisher", "test-stack-default", "test-stack")

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create error")
	})

	t.Run("success - schedule already exists", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Setup mocks
		mockScheduleClient := NewMockScheduleClient(ctrl)
		mockHandle := NewMockScheduleHandle(ctrl)
		mockClient := NewMockClient(ctrl)

		// Expectations
		mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()
		mockScheduleClient.EXPECT().GetHandle(ctx, "test-stack-outbox-publisher").Return(mockHandle)
		desc := &client.ScheduleDescription{}
		mockHandle.EXPECT().Describe(ctx).Return(desc, nil)

		activity := &Activities{
			temporalClient: mockClient,
		}

		// Execute activity
		err := activity.CreateOutboxPublisherSchedule(ctx, "test-stack-outbox-publisher", "test-stack-default", "test-stack")

		// Verify
		require.NoError(t, err)
	})

	t.Run("success - concurrent create already exists is ok", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockScheduleClient := NewMockScheduleClient(ctrl)
		mockHandle := NewMockScheduleHandle(ctrl)
		mockClient := NewMockClient(ctrl)

		mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()
		mockScheduleClient.EXPECT().GetHandle(ctx, "test-stack-outbox-publisher").Return(mockHandle)
		mockHandle.EXPECT().Describe(ctx).Return(nil, serviceerror.NewNotFound("not found"))
		mockScheduleClient.EXPECT().Create(ctx, gomock.Any()).Return(nil, serviceerror.NewAlreadyExists("exists"))

		activity := &Activities{temporalClient: mockClient}
		err := activity.CreateOutboxPublisherSchedule(ctx, "test-stack-outbox-publisher", "test-stack-default", "test-stack")
		require.NoError(t, err)
	})

	t.Run("error - describe fails with non not-found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockScheduleClient := NewMockScheduleClient(ctrl)
		mockHandle := NewMockScheduleHandle(ctrl)
		mockClient := NewMockClient(ctrl)

		mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()
		mockScheduleClient.EXPECT().GetHandle(ctx, "test-stack-outbox-publisher").Return(mockHandle)
		unexpected := fmt.Errorf("boom")
		mockHandle.EXPECT().Describe(ctx).Return(nil, unexpected)
		// Create should NOT be called

		activity := &Activities{temporalClient: mockClient}
		err := activity.CreateOutboxPublisherSchedule(ctx, "test-stack-outbox-publisher", "test-stack-default", "test-stack")
		require.Error(t, err)
		assert.ErrorContains(t, err, "describe schedule")
	})
}

func TestCreateOutboxPublisherSchedule_ScheduleOptions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup mocks
	mockScheduleClient := NewMockScheduleClient(ctrl)
	mockHandle := NewMockScheduleHandle(ctrl)
	mockClient := NewMockClient(ctrl)

	// Capture the schedule options
	var capturedOptions client.ScheduleOptions

	// Expectations
	mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()
	mockScheduleClient.EXPECT().GetHandle(ctx, "test-stack-outbox-publisher").Return(mockHandle)
	mockHandle.EXPECT().Describe(ctx).Return(nil, serviceerror.NewNotFound("not found"))
	mockScheduleClient.EXPECT().Create(ctx, gomock.Any()).Do(func(_ context.Context, opts client.ScheduleOptions) {
		capturedOptions = opts
	}).Return(mockHandle, nil)

	activity := &Activities{
		temporalClient: mockClient,
	}

	// Execute activity
	err := activity.CreateOutboxPublisherSchedule(ctx, "test-stack-outbox-publisher", "test-stack-default", "test-stack")
	require.NoError(t, err)

	// Verify schedule options
	assert.Equal(t, "test-stack-outbox-publisher", capturedOptions.ID)
	assert.NotNil(t, capturedOptions.Action)
	assert.Equal(t, "test-stack", capturedOptions.SearchAttributes["Stack"])
	act, ok := capturedOptions.Action.(*client.ScheduleWorkflowAction)
	require.True(t, ok, "expected ScheduleWorkflowAction")
	assert.Equal(t, "test-stack-default", act.TaskQueue)

}
