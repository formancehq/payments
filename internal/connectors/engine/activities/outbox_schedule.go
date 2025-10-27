package activities

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const OUTBOX_POLLING_PERIOD = 5 * time.Second

func (a Activities) CreateOutboxPublisherSchedule(ctx context.Context, stack string) error {
	scheduleID := fmt.Sprintf("%s-outbox-publisher", stack)

	// Check if schedule already exists
	_, err := a.temporalClient.ScheduleClient().GetHandle(ctx, scheduleID).Describe(ctx)
	if err == nil {
		// Schedule already exists, no need to create it
		return nil
	}

	// Create the schedule
	_, err = a.temporalClient.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID: scheduleID,
		Spec: client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{
				{
					Every: OUTBOX_POLLING_PERIOD, // Poll every 5 seconds
				},
			},
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        scheduleID,
			Workflow:  "OutboxPublisher",
			Args:      []interface{}{}, // No arguments needed
			TaskQueue: "payments",      // Use the default task queue
		},
		Overlap:            enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		TriggerImmediately: true,
		SearchAttributes: map[string]interface{}{
			"Stack": stack,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to create outbox publisher schedule: %w", err)
	}

	return nil
}

var CreateOutboxPublisherScheduleActivity = Activities{}.CreateOutboxPublisherSchedule

func CreateOutboxPublisherSchedule(ctx workflow.Context, stack string) error {
	return executeActivity(ctx, CreateOutboxPublisherScheduleActivity, nil, stack)
}
