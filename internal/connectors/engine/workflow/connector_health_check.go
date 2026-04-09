package workflow

import (
	"strings"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/sdk/workflow"
)

const RunConnectorHealthCheck = "ConnectorHealthCheck"

var fetchCapabilities = []models.Capability{
	models.CAPABILITY_FETCH_ACCOUNTS,
	models.CAPABILITY_FETCH_PAYMENTS,
	models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
	models.CAPABILITY_FETCH_BALANCES,
	models.CAPABILITY_FETCH_OTHERS,
}

type ConnectorHealthCheck struct {
	ConnectorID models.ConnectorID
	NextCursor  *string
}

func (w Workflow) runConnectorHealthCheck(ctx workflow.Context, req ConnectorHealthCheck) error {
	if err := w.createInstance(ctx, req.ConnectorID); err != nil {
		return errors.Wrap(err, "creating instance")
	}
	err := w.connectorHealthCheck(ctx, req)
	return w.terminateInstance(ctx, req.ConnectorID, err)
}

func (w Workflow) connectorHealthCheck(ctx workflow.Context, req ConnectorHealthCheck) error {
	connector, err := activities.StorageConnectorsGet(infiniteRetryContext(ctx), req.ConnectorID)
	if err != nil {
		return err
	}

	cursor := req.NextCursor

	for {
		result, err := activities.StorageInstancesListSchedulesAboveErrorThreshold(infiniteRetryContext(ctx), req.ConnectorID, cursor)
		if err != nil {
			return err
		}

		var toPause []models.Instance
		for _, instance := range result.Data {
			// skip instances that predate the last connector config update — the
			// config change may have resolved the issue
			if connector.UpdatedAt != nil && !instance.CreatedAt.After(*connector.UpdatedAt) {
				continue
			}
			// we only want to pause schedules related to fetching connector data;
			for _, capability := range fetchCapabilities {
				prefix := fetchNextWorkflowScheduleID(w.stack, req.ConnectorID.String(), capability.String(), nil)
				if strings.HasPrefix(instance.ScheduleID, prefix) {
					toPause = append(toPause, instance)
					break
				}
			}
		}

		if len(toPause) > 0 {
			if err := activities.TemporalSchedulesPause(infiniteRetryContext(ctx), toPause); err != nil {
				return err
			}
		}

		if !result.HasMore {
			break
		}

		cursor = &result.Next

		if w.shouldContinueAsNew(ctx) {
			return workflow.NewContinueAsNewError(ctx, RunConnectorHealthCheck, ConnectorHealthCheck{
				ConnectorID: req.ConnectorID,
				NextCursor:  cursor,
			})
		}
	}

	return nil
}
