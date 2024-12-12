package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

type UninstallConnector struct {
	ConnectorID       models.ConnectorID
	DefaultWorkerName string
}

func (w Workflow) runUninstallConnector(
	ctx workflow.Context,
	uninstallConnector UninstallConnector,
) error {
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
		uninstallConnector,
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
		defer wg.Done()
		_, err := activities.PluginUninstallConnector(infiniteRetryContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StorageEventsSentDelete(infiniteRetryContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StorageSchedulesDeleteFromConnectorID(infiniteRetryContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StorageInstancesDelete(infiniteRetryContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StorageConnectortTasksTreeDelete(infiniteRetryContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StorageTasksDeleteFromConnectorID(infiniteRetryContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StorageBankAccountsDeleteRelatedAccounts(infiniteRetryContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StorageAccountsDelete(infiniteRetryContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StoragePaymentsDelete(infiniteRetryContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StorageStatesDelete(infiniteRetryContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StorageWebhooksConfigsDelete(infiniteRetryContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StorageWebhooksDelete(infiniteRetryContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StoragePoolsRemoveAccountsFromConnectorID(infiniteRetryContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Wait(ctx)
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	err := activities.StorageConnectorsDelete(infiniteRetryContext(ctx), uninstallConnector.ConnectorID)
	if err != nil {
		return err
	}

	if err := w.plugins.UnregisterPlugin(uninstallConnector.ConnectorID); err != nil {
		return err
	}

	return nil
}

const RunUninstallConnector = "UninstallConnector"
