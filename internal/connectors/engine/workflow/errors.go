package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func (w Workflow) pluginError(
	ctx workflow.Context,
	connectorID models.ConnectorID,
	err error,
) error {
	if err == nil {
		return nil
	}

	temporalErr, ok := err.(*temporal.ApplicationError)
	if !ok {
		return err
	}

	switch temporalErr.Type() {
	case activities.ErrTypeNotYetInstalled:
		if errInstall := w.runInstallConnector(
			ctx,
			InstallConnector{
				ConnectorID: connectorID,
			},
		); errInstall != nil {
			workflow.GetLogger(ctx).Error("installing connector", "error", errInstall)
		}

		// return the original error since it's a retryable error and temporal
		// will retry the plugin call. It should work now that we have installed
		// the plugin.
		return err
	default:
		// Nothing to do
		return err
	}
}
