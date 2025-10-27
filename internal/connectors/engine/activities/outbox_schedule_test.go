package activities

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		mockHandle.EXPECT().Describe(ctx).Return(nil, errors.New("not found"))
		mockScheduleClient.EXPECT().Create(ctx, gomock.Any()).Return(mockHandle, nil)

		activity := &Activities{
			temporalClient: mockClient,
		}

		// Execute activity
		err := activity.CreateOutboxPublisherSchedule(ctx, "test-stack")

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
		mockHandle.EXPECT().Describe(ctx).Return(nil, errors.New("not found"))
		mockScheduleClient.EXPECT().Create(ctx, gomock.Any()).Return(nil, errors.New("create error"))

		activity := &Activities{
			temporalClient: mockClient,
		}

		// Execute activity
		err := activity.CreateOutboxPublisherSchedule(ctx, "test-stack")

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
		err := activity.CreateOutboxPublisherSchedule(ctx, "test-stack")

		// Verify
		require.NoError(t, err)
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
	mockHandle.EXPECT().Describe(ctx).Return(nil, errors.New("not found"))
	mockScheduleClient.EXPECT().Create(ctx, gomock.Any()).Do(func(_ context.Context, opts client.ScheduleOptions) {
		capturedOptions = opts
	}).Return(mockHandle, nil)

	activity := &Activities{
		temporalClient: mockClient,
	}

	// Execute activity
	err := activity.CreateOutboxPublisherSchedule(ctx, "test-stack")
	require.NoError(t, err)

	// Verify basic schedule options
	assert.Equal(t, "test-stack-outbox-publisher", capturedOptions.ID)
	assert.NotNil(t, capturedOptions.Action)
}
