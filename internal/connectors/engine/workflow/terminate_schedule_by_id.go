package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type TerminateScheduleByID struct {
	ConnectorID models.ConnectorID
	ScheduleID  string
}

func (w Workflow) runTerminateScheduleByID(
	ctx workflow.Context,
	terminateSchedule TerminateScheduleByID,
) error {
	schedule, err := activities.StorageSchedulesGet(infiniteRetryContext(ctx), terminateSchedule.ConnectorID, terminateSchedule.ScheduleID)
	if err != nil {
		return err
	}

	if err := activities.TemporalScheduleDelete(
		infiniteRetryContext(ctx),
		schedule.ID,
	); err != nil {
		return fmt.Errorf("failed to delete schedule %q from temporal: %w", schedule.ID, err)
	}

	if err := activities.StorageSchedulesDelete(infiniteRetryContext(ctx), schedule.ID); err != nil {
		return fmt.Errorf("failed to delete schedule %q from storage: %w", schedule.ID, err)
	}
	return nil
}

const RunTerminateScheduleByID = "TerminateScheduleByID"
