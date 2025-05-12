package workflow

import (
	"fmt"
	"strings"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type HandleWebhooks struct {
	ConnectorID models.ConnectorID
	URLPath     string
	Webhook     models.Webhook
}

func (w Workflow) runHandleWebhooks(
	ctx workflow.Context,
	handleWebhooks HandleWebhooks,
) error {
	configs, err := activities.StorageWebhooksConfigsGet(
		infiniteRetryContext(ctx),
		handleWebhooks.ConnectorID,
	)
	if err != nil {
		return fmt.Errorf("getting webhook configs: %w", err)
	}

	var config *models.WebhookConfig
	for _, c := range configs {
		if !strings.Contains(handleWebhooks.URLPath, c.URLPath) {
			continue
		}

		config = &c
		break
	}

	if config == nil {
		return temporal.NewNonRetryableApplicationError("webhook config not found", "NOT_FOUND", errors.New("webhook config not found"))
	}

	err = activities.StorageWebhooksStore(infiniteRetryContext(ctx), handleWebhooks.Webhook)
	if err != nil {
		return fmt.Errorf("storing webhook: %w", err)
	}

	resp, err := activities.PluginTranslateWebhook(
		infiniteRetryContext(ctx),
		handleWebhooks.ConnectorID,
		models.TranslateWebhookRequest{
			Name: config.Name,
			Webhook: models.PSPWebhook{
				BasicAuth:   handleWebhooks.Webhook.BasicAuth,
				QueryValues: handleWebhooks.Webhook.QueryValues,
				Headers:     handleWebhooks.Webhook.Headers,
				Body:        handleWebhooks.Webhook.Body,
			},
			Config: config,
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
					WorkflowID:            fmt.Sprintf("store-webhook-%s-%s-%s", w.stack, handleWebhooks.ConnectorID.String(), response.IdempotencyKey),
					TaskQueue:             w.getDefaultTaskQueue(),
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

const RunHandleWebhooks = "RunHandleWebhooks"

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
		accounts, err := models.FromPSPAccounts(
			[]models.PSPAccount{*storeWebhookTranslation.Account},
			models.ACCOUNT_TYPE_INTERNAL,
			storeWebhookTranslation.ConnectorID,
		)
		if err != nil {
			return temporal.NewNonRetryableApplicationError(
				"failed to translate accounts",
				ErrValidation,
				err,
			)
		}

		err = activities.StorageAccountsStore(
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
		accounts, err := models.FromPSPAccounts(
			[]models.PSPAccount{*storeWebhookTranslation.ExternalAccount},
			models.ACCOUNT_TYPE_EXTERNAL,
			storeWebhookTranslation.ConnectorID,
		)
		if err != nil {
			return temporal.NewNonRetryableApplicationError(
				"failed to translate accounts",
				ErrValidation,
				err,
			)
		}

		err = activities.StorageAccountsStore(
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
		payments, err := models.FromPSPPayments(
			[]models.PSPPayment{*storeWebhookTranslation.Payment},
			storeWebhookTranslation.ConnectorID,
		)
		if err != nil {
			return temporal.NewNonRetryableApplicationError(
				"failed to translate psp payments",
				ErrValidation,
				err,
			)
		}

		err = activities.StoragePaymentsStore(
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
					TaskQueue:         w.getDefaultTaskQueue(),
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

const RunStoreWebhookTranslation = "RunStoreWebhookTranslation"
