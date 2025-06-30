package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/engine/utils"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

type CreateWebhooks struct {
	ConnectorID models.ConnectorID
	Config      models.Config
	FromPayload *FromPayload
}

func (w Workflow) runCreateWebhooks(
	ctx workflow.Context,
	createWebhooks CreateWebhooks,
	nextTasks []models.ConnectorTaskTree,
) error {
	return w.createWebhooks(ctx, createWebhooks, nextTasks)
}

func (w Workflow) createWebhooks(
	ctx workflow.Context,
	createWebhooks CreateWebhooks,
	nextTasks []models.ConnectorTaskTree,
) error {
	webhookBaseURL, err := utils.GetWebhookBaseURL(w.stackPublicURL, createWebhooks.ConnectorID)
	if err != nil {
		return fmt.Errorf("joining webhook base URL: %w", err)
	}

	resp, err := activities.PluginCreateWebhooks(
		infiniteRetryContext(ctx),
		createWebhooks.ConnectorID,
		models.CreateWebhooksRequest{
			WebhookBaseUrl: webhookBaseURL,
			ConnectorID:    createWebhooks.ConnectorID.String(),
			FromPayload:    createWebhooks.FromPayload.GetPayload(),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create webhooks: %w", err)
	}

	if len(resp.Configs) > 0 {
		configs := make([]models.WebhookConfig, 0, len(resp.Configs))
		for _, c := range resp.Configs {
			configs = append(configs, models.WebhookConfig{
				Name:        c.Name,
				ConnectorID: createWebhooks.ConnectorID,
				URLPath:     c.URLPath,
				Metadata:    c.Metadata,
			})
		}
		err = activities.StorageWebhooksConfigsStore(
			infiniteRetryContext(ctx),
			configs,
		)
		if err != nil {
			return fmt.Errorf("storing webhooks: %w", err)
		}
	}

	for _, other := range resp.Others {
		if err := workflow.ExecuteChildWorkflow(
			workflow.WithChildOptions(
				ctx,
				workflow.ChildWorkflowOptions{
					TaskQueue:         w.getDefaultTaskQueue(),
					ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
					SearchAttributes: map[string]interface{}{
						SearchAttributeStack: w.stack,
					},
				},
			),
			Run,
			createWebhooks.Config,
			createWebhooks.ConnectorID,
			&FromPayload{
				ID:      other.ID,
				Payload: other.Other,
			},
			nextTasks,
		).Get(ctx, nil); err != nil {
			return fmt.Errorf("running next workflow: %w", err)
		}
	}

	return nil
}

const RunCreateWebhooks = "RunCreateWebhooks"
