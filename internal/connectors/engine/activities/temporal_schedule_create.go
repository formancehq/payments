package activities

import (
	"context"
	"errors"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type ScheduleCreateOptions struct {
	ScheduleID         string
	Interval           *client.ScheduleIntervalSpec
	Action             client.ScheduleWorkflowAction
	Overlap            enums.ScheduleOverlapPolicy
	Jitter             time.Duration
	TriggerImmediately bool
	SearchAttributes   map[string]interface{}
}

func (a Activities) TemporalScheduleCreate(ctx context.Context, options ScheduleCreateOptions) error {
	// Guard against a no-op schedule: nil Interval with TriggerImmediately false
	// produces a schedule that never fires. Such a schedule has no legitimate
	// use and indicates a caller bug — fail loudly instead of silently
	// registering dead state in Temporal.
	if options.Interval == nil && !options.TriggerImmediately {
		return temporal.NewNonRetryableApplicationError(
			"schedule would never fire: either set Interval (periodic) or TriggerImmediately (one-shot)",
			"invalidScheduleOptions",
			nil,
		)
	}

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

	spec := client.ScheduleSpec{
		Jitter: options.Jitter,
	}
	if options.Interval != nil {
		spec.Intervals = []client.ScheduleIntervalSpec{*options.Interval}
	}

	_, err := a.temporalClient.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID:                 options.ScheduleID,
		Spec:               spec,
		Action:             &options.Action,
		Overlap:            options.Overlap,
		TriggerImmediately: options.TriggerImmediately,
		SearchAttributes:   options.SearchAttributes,
	})
	if err != nil {
		// When triggering immediately or if a workflow with the same ID already exists,
		// Temporal may return either AlreadyExists (schedule exists) or
		// WorkflowExecutionAlreadyStarted (the workflow action with same ID already exists),
		// or the SDK sentinel error temporal.ErrScheduleAlreadyRunning when a schedule with the same ID
		// is already registered. All these cases should be treated as success as the desired state is achieved.
		var already *serviceerror.AlreadyExists
		var wfAlreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &wfAlreadyStarted) || errors.As(err, &already) {
			// Workflow already started with the same ID, treat as success
			return nil
		}
		if errors.Is(err, temporal.ErrScheduleAlreadyRunning) {
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
