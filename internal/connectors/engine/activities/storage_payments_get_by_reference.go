package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentsGetByReference(ctx context.Context, reference string, connectorID models.ConnectorID) (*models.Payment, error) {
	pi, err := a.storage.PaymentsGetByReference(ctx, reference, connectorID)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return pi, nil
}

var StoragePaymentsGetByReferenceActivity = Activities{}.StoragePaymentsGetByReference

func StoragePaymentsGetByReference(ctx workflow.Context, reference string, connectorID models.ConnectorID) (*models.Payment, error) {
	var result models.Payment
	err := executeActivity(ctx, StoragePaymentsGetByReferenceActivity, &result, reference, connectorID)
	return &result, err
}
