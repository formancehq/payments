package activities

import (
	"context"
	"errors"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
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

func (a Activities) TemporalScheduleCreate(ctx context.Context, options ScheduleCreateOptions) error {
	attributes := make([]temporal.SearchAttributeUpdate, 0, len(options.SearchAttributes))
	for key, value := range options.SearchAttributes {
		v, ok := value.(string)
		if !ok {
			continue
		}

		attributes = append(attributes,
			temporal.NewSearchAttributeKeyKeyword(key).ValueSet(v),
		)
	}
	options.Action.TypedSearchAttributes = temporal.NewSearchAttributes(attributes...)

	_, err := a.temporalClient.ScheduleClient().Create(ctx, client.ScheduleOptions{
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
		if errors.Is(err, temporal.ErrScheduleAlreadyRunning) {
			// No need to do anything if schedule is already running
			return nil
		}
		return err
	}

	return nil
}

var TemporalScheduleCreateActivity = Activities{}.TemporalScheduleCreate

func TemporalScheduleCreate(ctx workflow.Context, options ScheduleCreateOptions) error {
	if err := executeActivity(ctx, TemporalScheduleCreateActivity, nil, options); err != nil {
		return err
	}
	return nil
}
