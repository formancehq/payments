package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageBankAccountsAddRelatedAccount(ctx context.Context, bankAccountID uuid.UUID, relatedAccount models.BankAccountRelatedAccount) error {
	return temporalStorageError(a.storage.BankAccountsAddRelatedAccount(ctx, bankAccountID, relatedAccount))
}

var StorageBankAccountsAddRelatedAccountActivity = Activities{}.StorageBankAccountsAddRelatedAccount

func StorageBankAccountsAddRelatedAccount(ctx workflow.Context, bankAccountID uuid.UUID, relatedAccount models.BankAccountRelatedAccount) error {
	return executeActivity(ctx, StorageBankAccountsAddRelatedAccountActivity, nil, bankAccountID, relatedAccount)
}
