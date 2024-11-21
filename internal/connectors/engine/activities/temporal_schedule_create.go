package activities

import (
	"context"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

type ScheduleCreateOptions struct {
	ScheduleID         string
	Interval           client.ScheduleIntervalSpec
	Action             client.ScheduleWorkflowAction
	Overlap            enums.ScheduleOverlapPolicy
	Jitter             time.Duration
	TriggerImmediately bool
	SearchAttributes   map[string]interface{}
}

func (a Activities) TemporalScheduleCreate(ctx context.Context, options ScheduleCreateOptions) (string, error) {
	handle, err := a.temporalClient.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID: options.ScheduleID,
		Spec: client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{options.Interval},
			Jitter:    options.Jitter,
		},
		Action:             &options.Action,
		Overlap:            options.Overlap,
		TriggerImmediately: options.TriggerImmediately,
		SearchAttributes:   options.SearchAttributes,
	})
	if err != nil {
		return "", err
	}
	return handle.GetID(), nil
}

var TemporalScheduleCreateActivity = Activities{}.TemporalScheduleCreate

func TemporalScheduleCreate(ctx workflow.Context, options ScheduleCreateOptions) (string, error) {
	var scheduleID string
	if err := executeActivity(ctx, TemporalScheduleCreateActivity, &scheduleID, options); err != nil {
		return "", err
	}
	return scheduleID, nil
}
