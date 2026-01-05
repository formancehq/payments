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

		w.connectors.Unload(installConnector.ConnectorID)

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
		infiniteRetryContext(ctx),
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

	// First, we need to get the connector to check if it is scheduled for deletion
	// because if it is, we don't need to run the next tasks

	// Fourth step: launch the workflow tree, do not wait for the result
	// by using the GetChildWorkflowExecution function that returns a future
	// which will be ready when the child workflow has successfully started.
	plugin, err := w.connectors.Get(installConnector.ConnectorID)
	if err != nil {
		return fmt.Errorf("getting connector: %w", err)
	}

	if plugin.IsScheduledForDeletion() {
		return nil
	}
	if IsRunNextTaskOptimizationsEnabled(ctx) {
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
			RunNextTasksV3_1,
			installConnector.ConnectorID,
			nil,
			[]models.ConnectorTaskTree(installResponse.Workflow),
		).GetChildWorkflowExecution().Get(ctx, nil); err != nil {
			if temporal.IsWorkflowExecutionAlreadyStartedError(err) {
				return nil
			}
			return errors.Wrap(err, "running next workflow")
		}
	} else {
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
			RunNextTasks, //nolint:staticcheck // ignore deprecation
			models.Config{},
			installConnector.ConnectorID,
			nil,
			[]models.ConnectorTaskTree(installResponse.Workflow),
		).GetChildWorkflowExecution().Get(ctx, nil); err != nil {
			if temporal.IsWorkflowExecutionAlreadyStartedError(err) {
				return nil
			}
			return errors.Wrap(err, "running next workflow")
		}
	}

	return nil
}

const RunInstallConnector = "InstallConnector"
