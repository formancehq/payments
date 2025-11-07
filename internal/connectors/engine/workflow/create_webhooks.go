package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/engine/utils"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
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

	connector, err := activities.StorageConnectorsGet(infiniteRetryContext(ctx), createWebhooks.ConnectorID)
	if err != nil {
		return fmt.Errorf("getting connector: %w", err)
	}

	if connector.ScheduledForDeletion {
		// avoid scheduling next tasks if connector is scheduled for deletion
		return nil
	}

	wg := workflow.NewWaitGroup(ctx)
	errChan := make(chan error, len(resp.Others)*2)
	for _, other := range resp.Others {
		o := other

		wg.Add(1)
		workflow.Go(ctx, func(ctx workflow.Context) {
			defer wg.Done()

			if err := w.runNextTasks(
				ctx,
				createWebhooks.Config,
				connector,
				&FromPayload{
					ID:      o.ID,
					Payload: o.Other,
				},
				nextTasks,
			); err != nil {
				errChan <- errors.Wrap(err, "running next tasks")
			}
		})
	}

	wg.Wait(ctx)
	close(errChan)
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

const RunCreateWebhooks = "RunCreateWebhooks"
