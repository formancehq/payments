package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const RunScheduleConnectorHealthCheck = "ScheduleConnectorHealthCheck"

type ScheduleConnectorHealthCheck struct {
	ConnectorID models.ConnectorID
}

func (w Workflow) runScheduleConnectorHealthCheck(ctx workflow.Context, req ScheduleConnectorHealthCheck) error {
	scheduleID := fmt.Sprintf("%s-%s-HEALTH_CHECK", w.stack, req.ConnectorID.String())

	if err := activities.StorageSchedulesStore(
		infiniteRetryContext(ctx),
		models.Schedule{
			ID:          scheduleID,
			ConnectorID: req.ConnectorID,
			CreatedAt:   workflow.Now(ctx).UTC(),
		},
	); err != nil {
		return err
	}

	return activities.TemporalScheduleCreate(
		infiniteRetryContext(ctx),
		activities.ScheduleCreateOptions{
			ScheduleID: scheduleID,
			Interval: client.ScheduleIntervalSpec{
				Every: w.healthCheckPollingPeriod,
			},
			Action: client.ScheduleWorkflowAction{
				ID:        scheduleID,
				Workflow:  RunConnectorHealthCheck,
				Args:      []interface{}{ConnectorHealthCheck{ConnectorID: req.ConnectorID}},
				TaskQueue: w.getDefaultTaskQueue(),
			},
			Overlap:            enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
			TriggerImmediately: false,
			SearchAttributes: map[string]interface{}{
				SearchAttributeScheduleID: scheduleID,
				SearchAttributeStack:      w.stack,
			},
		},
	)
}
