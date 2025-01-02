package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type ResetConnector struct {
	ConnectorID       models.ConnectorID
	DefaultWorkerName string
	TaskID            models.TaskID
}

func (w Workflow) runResetConnector(
	ctx workflow.Context,
	resetConnector ResetConnector,
) error {
	newConnectorID, err := w.resetConnector(ctx, resetConnector)
	if err != nil {
		if errUpdateTask := w.updateTasksError(
			ctx,
			resetConnector.TaskID,
			nil,
			err,
		); errUpdateTask != nil {
			return errUpdateTask
		}

		return err
	}

	return w.updateTaskSuccess(
		ctx,
		resetConnector.TaskID,
		nil,
		newConnectorID.String(),
	)
}

func (w Workflow) resetConnector(
	ctx workflow.Context,
	resetConnector ResetConnector,
) (*models.ConnectorID, error) {
	connector, err := activities.StorageConnectorsGet(
		infiniteRetryContext(ctx),
		resetConnector.ConnectorID,
	)
	if err != nil {
		return nil, err
	}

	err = activities.StorageConnectorsScheduleForDeletion(
		infiniteRetryContext(ctx),
		resetConnector.ConnectorID,
	)
	if err != nil {
		return nil, err
	}

	if err := workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(
			ctx,
			workflow.ChildWorkflowOptions{
				TaskQueue:         resetConnector.DefaultWorkerName,
				ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
				SearchAttributes: map[string]interface{}{
					SearchAttributeStack: w.stack,
				},
			},
		),
		RunUninstallConnector,
		UninstallConnector{
			ConnectorID:       resetConnector.ConnectorID,
			DefaultWorkerName: resetConnector.DefaultWorkerName,
		},
	).Get(ctx, nil); err != nil {
		return nil, fmt.Errorf("uninstall connector: %w", err)
	}

	config := models.DefaultConfig()
	if err := json.Unmarshal(connector.Config, &config); err != nil {
		return nil, temporal.NewNonRetryableApplicationError("unmarshal config", "INVALID_CONFIG", err)
	}

	// We need to change the connector ID to a new one, otherwise, we will
	// have some conflicts with temporal and previous workflows related to the
	// previous connector ID.
	newConnector := models.Connector{
		ID: models.ConnectorID{
			Reference: uuid.New(),
			Provider:  connector.Provider,
		},
		Name:                 connector.Name,
		CreatedAt:            workflow.Now(ctx),
		Provider:             connector.Provider,
		ScheduledForDeletion: false,
		Config:               connector.Config,
	}

	if err := activities.StorageConnectorsStore(
		infiniteRetryContext(ctx),
		newConnector,
	); err != nil {
		return nil, err
	}

	if err := workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(
			ctx,
			workflow.ChildWorkflowOptions{
				WorkflowID:            fmt.Sprintf("install-%s-%s", w.stack, newConnector.ID.String()),
				TaskQueue:             w.getConnectorTaskQueue(newConnector.ID),
				WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
				ParentClosePolicy:     enums.PARENT_CLOSE_POLICY_ABANDON,
				SearchAttributes: map[string]interface{}{
					SearchAttributeStack: w.stack,
				},
			},
		),
		RunInstallConnector,
		InstallConnector{
			ConnectorID: resetConnector.ConnectorID,
			Config:      config,
		},
	).Get(ctx, nil); err != nil {
		return nil, fmt.Errorf("install connector: %w", err)
	}

	if err := workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(
			ctx,
			workflow.ChildWorkflowOptions{
				TaskQueue:         resetConnector.DefaultWorkerName,
				ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
				SearchAttributes: map[string]interface{}{
					SearchAttributeStack: w.stack,
				},
			},
		),
		RunSendEvents,
		SendEvents{
			ConnectorReset: &resetConnector.ConnectorID,
		},
	).Get(ctx, nil); err != nil {
		return nil, err
	}

	return &newConnector.ID, nil
}

const RunResetConnector = "ResetConnector"
