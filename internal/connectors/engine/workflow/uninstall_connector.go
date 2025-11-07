package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/pkg/errors"
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
			if errUpdateTask := w.updateTasksError(
				ctx,
				*uninstallConnector.TaskID,
				nil,
				err,
			); errUpdateTask != nil {
				return errUpdateTask
			}
		}

		return err
	}

	if uninstallConnector.TaskID == nil {
		return nil
	}

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
	webhooksConfigs, err := activities.StorageWebhooksConfigsGet(
		infiniteRetryContext(ctx),
		uninstallConnector.ConnectorID,
	)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
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
		_, err = activities.PluginUninstallConnector(infiniteRetryContext(ctx), uninstallConnector.ConnectorID, configs)
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
		err := activities.StorageAccountsDeleteFromConnectorID(infiniteRetryContext(ctx), uninstallConnector.ConnectorID)
		errChan <- err
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()
		err := activities.StoragePaymentsDeleteFromConnectorID(infiniteRetryContext(ctx), uninstallConnector.ConnectorID)
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

	err = activities.StorageConnectorsDelete(infiniteRetryContext(ctx), uninstallConnector.ConnectorID)
	if err != nil {
		return err
	}

	w.connectors.Unload(uninstallConnector.ConnectorID)

	return nil
}

const RunUninstallConnector = "UninstallConnector"
