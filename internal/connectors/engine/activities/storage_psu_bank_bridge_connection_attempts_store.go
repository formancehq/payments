package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePSUBankBridgeConnectionAttemptsStore(ctx context.Context, from models.PSUBankBridgeConnectionAttempt) error {
	return temporalStorageError(a.storage.PSUBankBridgeConnectionAttemptsUpsert(ctx, from))
}

var StoragePSUBankBridgeConnectionAttemptsStoreActivity = Activities{}.StoragePSUBankBridgeConnectionAttemptsStore

func StoragePSUBankBridgeConnectionAttemptsStore(ctx workflow.Context, from models.PSUBankBridgeConnectionAttempt) error {
	return executeActivity(ctx, StoragePSUBankBridgeConnectionAttemptsStoreActivity, nil, from)
}
