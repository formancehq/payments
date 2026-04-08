package workflow

import (
	"strings"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/sdk/workflow"
)

const RunConnectorHealthCheck = "ConnectorHealthCheck"

var fetchCapabilities = []string{
	models.CAPABILITY_FETCH_ACCOUNTS.String(),
	models.CAPABILITY_FETCH_PAYMENTS.String(),
	models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS.String(),
	models.CAPABILITY_FETCH_BALANCES.String(),
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
	cursor := req.NextCursor

	for {
		result, err := activities.StorageInstancesGetErrors(infiniteRetryContext(ctx), req.ConnectorID, cursor)
		if err != nil {
			return err
		}

		var toPause []models.Instance
		for _, instance := range result.Data {
			// we only want to pause schedules related to fetching connector data
			for _, capability := range fetchCapabilities {
				if strings.Contains(instance.ScheduleID, capability) {
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
