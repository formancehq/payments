package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type InstallConnector struct {
	ConnectorID models.ConnectorID
	Config      models.Config
}

func (w Workflow) runInstallConnector(
	ctx workflow.Context,
	installConnector InstallConnector,
) error {
	if errInstall := w.installConnector(ctx, installConnector); errInstall != nil {
		// In that case we don't want the connector to still be present in the
		// database, so we remove it
		if err := activities.StorageConnectorsDelete(
			infiniteRetryContext(ctx),
			installConnector.ConnectorID,
		); err != nil {
			return fmt.Errorf("failed to delete connector: %w", err)
		}

		w.plugins.UnregisterPlugin(installConnector.ConnectorID)

		return errInstall
	}

	return nil
}

func (w Workflow) installConnector(
	ctx workflow.Context,
	installConnector InstallConnector,
) error {
	// Second step: install the connector via the plugin and get the list of
	// capabilities and the workflow of polling data
	installResponse, err := activities.PluginInstallConnector(
		// disable retries as grpc plugin boot command cannot be run more than once by the go-plugin client
		// this also causes API install calls to fail immediately which is more desirable in the case that a plugin is timing out or not compiled correctly
		maximumAttemptsRetryContext(ctx, 1),
		installConnector.ConnectorID,
	)
	if err != nil {
		return errors.Wrap(err, "failed to install connector")
	}

	// Third step: store the workflow of the connector
	err = activities.StorageConnectorTasksTreeStore(infiniteRetryContext(ctx), installConnector.ConnectorID, installResponse.Workflow)
	if err != nil {
		return errors.Wrap(err, "failed to store tasks tree")
	}

	// Fifth step: launch the workflow tree, do not wait for the result
	// by using the GetChildWorkflowExecution function that returns a future
	// which will be ready when the child workflow has successfully started.
	if err := workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(
			ctx,
			workflow.ChildWorkflowOptions{
				WorkflowID:            fmt.Sprintf("run-tasks-%s-%s", w.stack, installConnector.ConnectorID.String()),
				WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
				TaskQueue:             w.getDefaultTaskQueue(),
				ParentClosePolicy:     enums.PARENT_CLOSE_POLICY_ABANDON,
				SearchAttributes: map[string]interface{}{
					SearchAttributeStack: w.stack,
				},
			},
		),
		Run,
		installConnector.Config,
		installConnector.ConnectorID,
		nil,
		[]models.ConnectorTaskTree(installResponse.Workflow),
	).GetChildWorkflowExecution().Get(ctx, nil); err != nil {
		if temporal.IsWorkflowExecutionAlreadyStartedError(err) {
			return nil
		} else {
			return errors.Wrap(err, "running next workflow")
		}
	}

	return nil
}

const RunInstallConnector = "InstallConnector"
