package activities

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) EventsSendBankAccount(ctx context.Context, bankAccount models.BankAccount) error {
	ba, err := a.events.NewEventSavedBankAccounts(bankAccount)
	if err != nil {
		return fmt.Errorf("failed to send bank account: %w", err)
	}
	return a.events.Publish(ctx, ba)
}

var EventsSendBankAccountActivity = Activities{}.EventsSendBankAccount

func EventsSendBankAccount(ctx workflow.Context, bankAccount models.BankAccount) error {
	return executeActivity(ctx, EventsSendBankAccountActivity, nil, bankAccount)
}
