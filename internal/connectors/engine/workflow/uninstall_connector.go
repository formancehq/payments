package workflow

import (
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

type UninstallConnector struct {
	ConnectorID       models.ConnectorID
	DefaultWorkerName string
	TaskID            *models.TaskID
}

func (w Workflow) runUninstallConnector(
	ctx workflow.Context,
	uninstallConnector UninstallConnector,
) error {
	err := w.uninstallConnector(ctx, uninstallConnector)
	if err != nil {
		if uninstallConnector.TaskID != nil {
			// Do not associate the task to the connector on error, as the connector
			// might already be deleted at this point, which would cause a foreign key
			// violation when persisting the task update.
			if errUpdateTask := w.updateTasksError(
				ctx,
				*uninstallConnector.TaskID,
				nil,
				err,
			); errUpdateTask != nil {
				return fmt.Errorf("failed to update task after uninstall error: %w (original error: %v)", errUpdateTask, err)
			}
		}

		return err
	}

	if uninstallConnector.TaskID == nil {
		return nil
	}

	// After successful uninstallation, the connector is deleted from storage.
	// To avoid foreign key violations when persisting the task update, do not
	// associate the task with the (now deleted) connector. We still record the
	// connector identifier in CreatedObjectID for traceability.
	return w.updateTaskSuccess(
		ctx,
		*uninstallConnector.TaskID,
		nil,
		uninstallConnector.ConnectorID.String(),
	)
}

func (w Workflow) uninstallConnector(
	ctx workflow.Context,
	uninstallConnector UninstallConnector,
) error {
	const startToFinishTimeoutForLongRunningActivities = 1 * time.Hour
	const heartbeatTimeoutForLongRunningActivities = 30 * time.Second

	webhooksConfigs, err := activities.StorageWebhooksConfigsGet(
		infiniteRetryContext(ctx),
		uninstallConnector.ConnectorID,
	)
	if err != nil {
		if !isStorageNotFoundError(err) {
			return err
		}

		webhooksConfigs = []models.WebhookConfig{}
	}

	// First, terminate all schedules in order to prevent any workflows
	// to be launched again.
	if err := workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(
			ctx,
			workflow.ChildWorkflowOptions{
				TaskQueue:         uninstallConnector.DefaultWorkerName,
				ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
				SearchAttributes: map[string]interface{}{
					SearchAttributeStack: w.stack,
				},
			},
		),
		RunTerminateSchedules,
		TerminateSchedules{
			ConnectorID:   uninstallConnector.ConnectorID,
			NextPageToken: "",
		},
	).Get(ctx, nil); err != nil {
		return fmt.Errorf("terminate schedules: %w", err)
	}

	// Since we can have lots of workflows running, we don't need to wait for
	// them to be terminated before proceeding with the uninstallation.
	if err := workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(
			ctx,
			workflow.ChildWorkflowOptions{
				TaskQueue:         uninstallConnector.DefaultWorkerName,
				ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
				SearchAttributes: map[string]interface{}{
					SearchAttributeStack: w.stack,
				},
			},
		),
		RunTerminateWorkflows,
		TerminateWorkflows{
			ConnectorID: uninstallConnector.ConnectorID,
		},
	).GetChildWorkflowExecution().Get(ctx, nil); err != nil {
		return fmt.Errorf("terminate workflows: %w", err)
	}

	wg := workflow.NewWaitGroup(ctx)
	errChan := make(chan error, 32)

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		configs := models.ToPSPWebhookConfigs(webhooksConfigs)
		defer wg.Done()
		_, err = activities.PluginUninstallConnector(infiniteRetryWithLongTimeoutContext(ctx), uninstallConnector.ConnectorID, configs)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		// Use heartbeat timeout to prevent timeout on large deletions with batching
		err := activities.StorageEventsSentDelete(
			infiniteRetryWithCustomStartToCloseAndHeartbeatContext(ctx, startToFinishTimeoutForLongRunningActivities, heartbeatTimeoutForLongRunningActivities),
			uninstallConnector.ConnectorID,
		)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		// Use heartbeat timeout to prevent timeout on large deletions with batching
		err := activities.StorageSchedulesDeleteFromConnectorID(
			infiniteRetryWithCustomStartToCloseAndHeartbeatContext(ctx, startToFinishTimeoutForLongRunningActivities, heartbeatTimeoutForLongRunningActivities),
			uninstallConnector.ConnectorID,
		)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		// Use heartbeat timeout to prevent timeout on large deletions with batching
		err := activities.StorageInstancesDelete(
			infiniteRetryWithCustomStartToCloseAndHeartbeatContext(ctx, startToFinishTimeoutForLongRunningActivities, heartbeatTimeoutForLongRunningActivities),
			uninstallConnector.ConnectorID,
		)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StorageConnectortTasksTreeDelete(infiniteRetryWithLongTimeoutContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StorageTasksDeleteFromConnectorID(infiniteRetryWithLongTimeoutContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StorageBankAccountsDeleteRelatedAccounts(infiniteRetryWithLongTimeoutContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		// Use heartbeat timeout to prevent timeout on large deletions with batching
		err := activities.StorageAccountsDeleteFromConnectorID(
			infiniteRetryWithCustomStartToCloseAndHeartbeatContext(ctx, startToFinishTimeoutForLongRunningActivities, heartbeatTimeoutForLongRunningActivities),
			uninstallConnector.ConnectorID,
		)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StoragePaymentsDeleteFromConnectorID(
			infiniteRetryWithCustomStartToCloseAndHeartbeatContext(ctx, startToFinishTimeoutForLongRunningActivities, heartbeatTimeoutForLongRunningActivities),
			uninstallConnector.ConnectorID,
		)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StorageStatesDelete(infiniteRetryWithLongTimeoutContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StorageWebhooksConfigsDelete(infiniteRetryWithLongTimeoutContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StorageWebhooksDelete(infiniteRetryWithLongTimeoutContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StoragePoolsRemoveAccountsFromConnectorID(infiniteRetryWithLongTimeoutContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Wait(ctx)
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	err = activities.StorageConnectorsDelete(infiniteRetryContext(ctx), uninstallConnector.ConnectorID)
	if err != nil {
		return err
	}

	w.connectors.Unload(uninstallConnector.ConnectorID)

	return nil
}

const RunUninstallConnector = "UninstallConnector"
