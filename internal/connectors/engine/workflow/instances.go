package workflow

import (
	"encoding/json"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/sdk/workflow"
)

var (
	errNotFromSchedule = errors.New("not from schedule")
)

func (w Workflow) createInstance(
	ctx workflow.Context,
	connectorID models.ConnectorID,
) error {
	info := workflow.GetInfo(ctx)

	scheduleID, err := getPaymentScheduleID(info)
	if err != nil {
		if errors.Is(err, errNotFromSchedule) {
			return nil
		}
		return err
	}

	instance := models.Instance{
		ID:          info.WorkflowExecution.ID,
		ScheduleID:  scheduleID,
		ConnectorID: connectorID,
		CreatedAt:   workflow.Now(ctx).UTC(),
		UpdatedAt:   workflow.Now(ctx).UTC(),
		Terminated:  false,
	}

	return activities.StorageInstancesStore(infiniteRetryContext(ctx), instance)
}

func (w Workflow) terminateInstance(
	ctx workflow.Context,
	connectorID models.ConnectorID,
	terminateError error,
) error {
	info := workflow.GetInfo(ctx)

	scheduleID, err := getPaymentScheduleID(info)
	if err != nil {
		if errors.Is(err, errNotFromSchedule) {
			return selectError(terminateError, nil)
		}
		w.logger.WithField("workflow_id", info.WorkflowExecution.ID).Errorf("failed to get payment schedule ID when terminating instance: %w", err)
		return selectError(terminateError, err)
	}

	var errMessage *string
	if terminateError != nil {
		errMessage = pointer.For(terminateError.Error())
	}

	now := workflow.Now(ctx).UTC()

	instance := models.Instance{
		ID:           info.WorkflowExecution.ID,
		ScheduleID:   scheduleID,
		ConnectorID:  connectorID,
		UpdatedAt:    now,
		Terminated:   true,
		TerminatedAt: &now,
		Error:        errMessage,
	}

	err = activities.StorageInstancesUpdate(infiniteRetryContext(ctx), instance)
	if err != nil {
		w.logger.WithField("workflow_id", info.WorkflowExecution.ID).Errorf("failed to update workflow instance: %w", err)
	}
	return selectError(terminateError, err)
}

func selectError(err1, err2 error) error {
	if err1 != nil {
		return err1
	}
	return err2
}

func getPaymentScheduleID(
	info *workflow.Info,
) (string, error) {
	attributes := info.SearchAttributes.GetIndexedFields() //nolint:staticcheck
	if attributes == nil {
		return "", errNotFromSchedule
	}

	v, ok := attributes[SearchAttributeScheduleID]
	if !ok || v == nil {
		return "", errNotFromSchedule
	}

	var scheduleID string
	if err := json.Unmarshal(v.Data, &scheduleID); err != nil {
		return "", errors.Wrap(err, "unmarshalling schedule ID")
	}

	return scheduleID, nil
}
