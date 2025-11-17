package activities

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const OUTBOX_POLLING_PERIOD = 5 * time.Second

func (a Activities) CreateOutboxPublisherSchedule(ctx context.Context, scheduleID, taskQueue, stack string) error {
	// Check if schedule already exists
	_, err := a.temporalClient.ScheduleClient().GetHandle(ctx, scheduleID).Describe(ctx)
	if err == nil {
		// Schedule already exists, no need to create it
		return nil
	}
	var notFoundErr *serviceerror.NotFound
	if !errors.As(err, &notFoundErr) {
		// Some other error while describing: fail fast
		return fmt.Errorf("describe schedule %s: %w", scheduleID, err)
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
			TaskQueue: taskQueue,
		},
		Overlap:            enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		TriggerImmediately: true,
		SearchAttributes: map[string]interface{}{
			"Stack": stack,
		},
	})

	if err != nil {
		var already *serviceerror.AlreadyExists
		if errors.As(err, &already) {
			// Created by concurrent caller, treat as success
			return nil
		}
		return fmt.Errorf("failed to create outbox publisher schedule: %w", err)
	}

	return nil
}

var CreateOutboxPublisherScheduleActivity = Activities{}.CreateOutboxPublisherSchedule

func CreateOutboxPublisherSchedule(ctx workflow.Context, scheduleID, taskQueue, stack string) error {
	return executeActivity(ctx, CreateOutboxPublisherScheduleActivity, nil, scheduleID, taskQueue, stack)
}
