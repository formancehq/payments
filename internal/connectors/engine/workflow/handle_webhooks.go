package workflow

import (
	"fmt"

	"github.com/formancehq/go-libs/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type HandleWebhooks struct {
	ConnectorID   models.ConnectorID
	WebhookConfig models.WebhookConfig
	Webhook       models.Webhook
}

func (w Workflow) runHandleWebhooks(
	ctx workflow.Context,
	handleWebhooks HandleWebhooks,
) error {
	err := activities.StorageWebhooksStore(infiniteRetryContext(ctx), handleWebhooks.Webhook)
	if err != nil {
		return fmt.Errorf("storing webhook: %w", err)
	}

	resp, err := activities.PluginTranslateWebhook(
		infiniteRetryContext(ctx),
		handleWebhooks.ConnectorID,
		models.TranslateWebhookRequest{
			Name: handleWebhooks.WebhookConfig.Name,
			Webhook: models.PSPWebhook{
				BasicAuth:   handleWebhooks.Webhook.BasicAuth,
				QueryValues: handleWebhooks.Webhook.QueryValues,
				Headers:     handleWebhooks.Webhook.Headers,
				Body:        handleWebhooks.Webhook.Body,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("translating webhook: %w", err)
	}

	for _, response := range resp.Responses {
		if err := workflow.ExecuteChildWorkflow(
			workflow.WithChildOptions(
				ctx,
				workflow.ChildWorkflowOptions{
					WorkflowID:            fmt.Sprintf("store-webhook-%s-%s", handleWebhooks.ConnectorID.String(), response.IdempotencyKey),
					TaskQueue:             handleWebhooks.ConnectorID.String(),
					ParentClosePolicy:     enums.PARENT_CLOSE_POLICY_ABANDON,
					WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
					SearchAttributes: map[string]interface{}{
						SearchAttributeStack: w.stack,
					},
				},
			),
			RunStoreWebhookTranslation,
			StoreWebhookTranslation{
				ConnectorID:     handleWebhooks.ConnectorID,
				Account:         response.Account,
				ExternalAccount: response.ExternalAccount,
				Payment:         response.Payment,
			},
		).Get(ctx, nil); err != nil {
			applicationError := &temporal.ApplicationError{}
			if errors.As(err, &applicationError) {
				if applicationError.Type() != "ChildWorkflowExecutionAlreadyStartedError" {
					return err
				}
			} else {
				return fmt.Errorf("storing webhook translation: %w", err)
			}
		}
	}

	return nil
}

var RunHandleWebhooks any

type StoreWebhookTranslation struct {
	ConnectorID     models.ConnectorID
	Account         *models.PSPAccount
	ExternalAccount *models.PSPAccount
	Payment         *models.PSPPayment
}

func (w Workflow) runStoreWebhookTranslation(
	ctx workflow.Context,
	storeWebhookTranslation StoreWebhookTranslation,
) error {
	var sendEvent *SendEvents
	if storeWebhookTranslation.Account != nil {
		accounts := models.FromPSPAccounts(
			[]models.PSPAccount{*storeWebhookTranslation.Account},
			models.ACCOUNT_TYPE_INTERNAL,
			storeWebhookTranslation.ConnectorID,
		)

		err := activities.StorageAccountsStore(
			infiniteRetryContext(ctx),
			accounts,
		)
		if err != nil {
			return fmt.Errorf("storing next accounts: %w", err)
		}

		sendEvent = &SendEvents{
			Account: pointer.For(accounts[0]),
		}
	}

	if storeWebhookTranslation.ExternalAccount != nil {
		accounts := models.FromPSPAccounts(
			[]models.PSPAccount{*storeWebhookTranslation.ExternalAccount},
			models.ACCOUNT_TYPE_EXTERNAL,
			storeWebhookTranslation.ConnectorID,
		)

		err := activities.StorageAccountsStore(
			infiniteRetryContext(ctx),
			accounts,
		)
		if err != nil {
			return fmt.Errorf("storing next accounts: %w", err)
		}

		sendEvent = &SendEvents{
			Account: pointer.For(accounts[0]),
		}
	}

	if storeWebhookTranslation.Payment != nil {
		payments := models.FromPSPPayments(
			[]models.PSPPayment{*storeWebhookTranslation.Payment},
			storeWebhookTranslation.ConnectorID,
		)
		err := activities.StoragePaymentsStore(
			infiniteRetryContext(ctx),
			payments,
		)
		if err != nil {
			return fmt.Errorf("storing next payments: %w", err)
		}

		sendEvent = &SendEvents{
			Payment: pointer.For(payments[0]),
		}
	}

	if sendEvent != nil {
		if err := workflow.ExecuteChildWorkflow(
			workflow.WithChildOptions(
				ctx,
				workflow.ChildWorkflowOptions{
					TaskQueue:         storeWebhookTranslation.ConnectorID.String(),
					ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
					SearchAttributes: map[string]interface{}{
						SearchAttributeStack: w.stack,
					},
				},
			),
			RunSendEvents,
			*sendEvent,
		).Get(ctx, nil); err != nil {
			return fmt.Errorf("sending events: %w", err)
		}
	}

	return nil
}

var RunStoreWebhookTranslation any

func init() {
	RunHandleWebhooks = Workflow{}.runHandleWebhooks
	RunStoreWebhookTranslation = Workflow{}.runStoreWebhookTranslation
}
