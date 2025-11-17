package client

import (
	"context"
	"fmt"
	"github.com/stripe/stripe-go/v79"
)

type EventCategory string

const (
	// Payouts
	EventCategoryPayoutCreated  EventCategory = "payout.created"
	EventCategoryPayoutFailed   EventCategory = "payout.failed"
	EventCategoryPayoutCanceled EventCategory = "payout.canceled"
	EventCategoryPayoutPaid     EventCategory = "payout.paid"
	EventCategoryPayoutUpdated  EventCategory = "payout.updated"

	// Transfers
	EventCategoryTransferCreated  EventCategory = "transfer.created"
	EventCategoryTransferReversed EventCategory = "transfer.reversed"
	EventCategoryTransferUpdated  EventCategory = "transfer.updated"

	// Accounts
	EventCategoryAccountUpdated         EventCategory = "account.updated"
	EventCategoryExternalAccountCreated EventCategory = "account.external_account.created"
	EventCategoryExternalAccountDeleted EventCategory = "account.external_account.deleted"
	EventCategoryExternalAccountUpdated EventCategory = "account.external_account.updated"

	// Balances
	EventCategoryBalanceAvailable EventCategory = "balance.available"

	//// Needed?
	EventCategoryPayoutReconciliationCompleted         EventCategory = "payout.reconciliation_completed"
	EventCategoryCustomerCashBalanceTransactionCreated EventCategory = "customer.cash_balance_transaction.created"
)

type CreateWebhookResponse struct {
	// Unique identifier for the object.
	WebhookEndpointID string
	// The endpoint's secret, used to verify webhook signatures.
	Secret string
}

func (p *client) CreateWebhook(_ context.Context, webhookURL string, connectorID string, event stripe.EventType) (CreateWebhookResponse, error) {
	params := &stripe.WebhookEndpointParams{
		EnabledEvents: []*string{
			stripe.String(string(event)),
		},
		URL: stripe.String(webhookURL),

		Description: stripe.String(fmt.Sprintf("Formance webhoook endpoint for event %s", string(event))),
		Metadata: map[string]string{
			"connector_id": connectorID,
		},
		Connect:    stripe.Bool(true),
		APIVersion: stripe.String(stripe.APIVersion),
	}

	endpoint, err := p.webhookEndpointClient.New(params)
	if err != nil {
		return CreateWebhookResponse{}, err
	}

	return CreateWebhookResponse{
		Secret:            endpoint.Secret,
		WebhookEndpointID: endpoint.ID,
	}, nil
}

func (p *client) DeleteWebhook(_ context.Context, webhookEndpointID string) error {
	_, err := p.webhookEndpointClient.Del(webhookEndpointID, nil)
	return err
}
