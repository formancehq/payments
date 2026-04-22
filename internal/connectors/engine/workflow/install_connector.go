package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
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
		// Capture whether the connector declared bootstrap tasks BEFORE we
		// Unload it — after Unload, connectors.Get(...) returns an error and
		// we can no longer inspect the plugin. If it declared bootstrap tasks,
		// the install branch may have created the one-shot schedule before
		// failing and we need to clean it up below.
		hasBootstrap := false
		if plugin, err := w.connectors.Get(installConnector.ConnectorID); err == nil {
			if p, ok := plugin.(models.PluginWithBootstrapOnInstall); ok {
				hasBootstrap = len(p.BootstrapOnInstall()) > 0
			}
		}

		// In that case we don't want the connector to still be present in the
		// database, so we remove it
		if err := activities.StorageConnectorsDelete(
			infiniteRetryContext(ctx),
			installConnector.ConnectorID,
		); err != nil {
			return fmt.Errorf("failed to delete connector: %w", err)
		}

		w.connectors.Unload(installConnector.ConnectorID)

		// Best-effort cleanup of the bootstrap schedule created during
		// installConnector. Errors are logged but do not override the primary
		// install error — matching the pattern in instances.go:55 and
		// instances.go:79.
		if hasBootstrap {
			bootstrapScheduleID := w.bootstrapScheduleID(installConnector.ConnectorID)
			if errDel := activities.TemporalScheduleDelete(infiniteRetryContext(ctx), bootstrapScheduleID); errDel != nil {
				w.logger.WithFields(map[string]any{
					"schedule_id": bootstrapScheduleID,
					"error":       errDel,
				}).Error("failed to delete bootstrap temporal schedule after install failure")
			}
			if errDel := activities.StorageSchedulesDelete(infiniteRetryContext(ctx), bootstrapScheduleID); errDel != nil {
				w.logger.WithFields(map[string]any{
					"schedule_id": bootstrapScheduleID,
					"error":       errDel,
				}).Error("failed to delete bootstrap storage schedule after install failure")
			}
		}

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

	// If the plugin declares bootstrap tasks, hand off to RunBootstrapTasks
	// instead of starting the periodic scheduler inline. RunBootstrapTasks
	// runs the declared tasks to completion and then starts the periodic
	// scheduler itself — so the periodic workflows only begin once the
	// plugin's initial data is in the database.
	//
	if p, ok := plugin.(models.PluginWithBootstrapOnInstall); ok {
		if taskTypes := p.BootstrapOnInstall(); len(taskTypes) > 0 {
			bootstrapScheduleID := w.bootstrapScheduleID(installConnector.ConnectorID)

			// One-shot schedule — registered to inherit the uninstall-cleanup path.
			// Must not be targeted by pause/unpause operations; those are only
			// meaningful for periodic schedules.
			if err := activities.StorageSchedulesStore(
				infiniteRetryContext(ctx),
				models.Schedule{
					ID:          bootstrapScheduleID,
					ConnectorID: installConnector.ConnectorID,
					CreatedAt:   workflow.Now(ctx).UTC(),
				},
			); err != nil {
				return errors.Wrap(err, "storing bootstrap schedule")
			}

			if err := activities.TemporalScheduleCreate(
				infiniteRetryContext(ctx),
				activities.ScheduleCreateOptions{
					ScheduleID: bootstrapScheduleID,
					Action: client.ScheduleWorkflowAction{
						ID:       bootstrapScheduleID,
						Workflow: RunBootstrapTasks,
						Args: []interface{}{
							BootstrapTasksRequest{
								ConnectorID: installConnector.ConnectorID,
								TaskTypes:   taskTypes,
								TaskTree:    []models.ConnectorTaskTree(installResponse.Workflow),
							},
						},
						TaskQueue: w.getDefaultTaskQueue(),
					},
					TriggerImmediately: true,
					SearchAttributes: map[string]interface{}{
						SearchAttributeScheduleID: bootstrapScheduleID,
						SearchAttributeStack:      w.stack,
					},
				},
			); err != nil {
				return errors.Wrap(err, "creating bootstrap schedule")
			}

			return w.scheduleConnectorHealthCheck(ctx, installConnector)
		}
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

	return w.scheduleConnectorHealthCheck(ctx, installConnector)
}

// scheduleConnectorHealthCheck launches the health check schedule without
// waiting for it to complete. GetChildWorkflowExecution waits only for the
// child to start, so any start-time error is returned while completion runs
// independently.
func (w Workflow) scheduleConnectorHealthCheck(
	ctx workflow.Context,
	installConnector InstallConnector,
) error {
	if err := workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID:            fmt.Sprintf("schedule-health-check-%s-%s", w.stack, installConnector.ConnectorID.String()),
			WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
			TaskQueue:             w.getDefaultTaskQueue(),
			ParentClosePolicy:     enums.PARENT_CLOSE_POLICY_ABANDON,
			SearchAttributes: map[string]interface{}{
				SearchAttributeStack: w.stack,
			},
		}),
		RunScheduleConnectorHealthCheck,
		ScheduleConnectorHealthCheck(installConnector),
	).GetChildWorkflowExecution().Get(ctx, nil); err != nil {
		if temporal.IsWorkflowExecutionAlreadyStartedError(err) {
			return nil
		}
		return errors.Wrap(err, "scheduling connector health check")
	}

	return nil
}

const RunInstallConnector = "InstallConnector"
