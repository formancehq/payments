package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/google/uuid"
	"go.temporal.io/api/enums/v1"
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
				TaskQueue:             resetConnector.DefaultWorkerName,
				ParentClosePolicy:     enums.PARENT_CLOSE_POLICY_ABANDON,
				SearchAttributes:      w.SearchAttributes(ctx, &resetConnector.ConnectorID),
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

	// We need to change the connector ID to a new one, otherwise, we will
	// have some conflicts with temporal and previous workflows related to the
	// previous connector ID.
	//
	// The reference must be generated through workflow.SideEffect: calling
	// uuid.New() directly in the workflow body is non-deterministic, so on replay
	// a different UUID would be produced and the install child WorkflowID below
	// (install-{stack}-{newConnector.ID}) would no longer match history, failing
	// the workflow task in a loop and leaving the reset stuck (EN-1093 / H10).
	var newReference uuid.UUID
	if err := workflow.SideEffect(ctx, func(workflow.Context) interface{} {
		return uuid.New()
	}).Get(&newReference); err != nil {
		return nil, fmt.Errorf("generating new connector reference: %w", err)
	}

	newConnector := models.Connector{
		ConnectorBase: models.ConnectorBase{
			ID: models.ConnectorID{
				Reference: newReference,
				Provider:  connector.Provider,
			},
			Name:      connector.Name,
			CreatedAt: workflow.Now(ctx),
			Provider:  connector.Provider,
		},
		ScheduledForDeletion: false,
		Config:               connector.Config,
	}

	if err := activities.StorageConnectorsStore(
		infiniteRetryContext(ctx),
		newConnector,
		&resetConnector.ConnectorID,
	); err != nil {
		return nil, err
	}

	if err := workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(
			ctx,
			workflow.ChildWorkflowOptions{
				WorkflowID:            fmt.Sprintf("install-%s-%s", w.stack, newConnector.ID.String()),
				TaskQueue:             w.getDefaultTaskQueue(),
				WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
				ParentClosePolicy:     enums.PARENT_CLOSE_POLICY_ABANDON,
				SearchAttributes:      w.SearchAttributes(ctx, &newConnector.ID),
			},
		),
		RunInstallConnector,
		InstallConnector{
			ConnectorID: newConnector.ID,
		},
	).Get(ctx, nil); err != nil {
		return nil, fmt.Errorf("install connector: %w", err)
	}

	return &newConnector.ID, nil
}

const RunResetConnector = "ResetConnector"
