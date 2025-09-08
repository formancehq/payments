package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type DeleteBankBridgeConnectionData struct {
	PSUID uuid.UUID

	FromConnectionID *DeleteBankBridgeConnectionDataFromConnectionID
	FromAccountID    *DeleteBankBridgeConnectionDataFromAccountID
	FromConnectorID  *DeleteBankBridgeConnectionDataFromConnectorID
}

type DeleteBankBridgeConnectionDataFromConnectionID struct {
	ConnectionID string
}

type DeleteBankBridgeConnectionDataFromAccountID struct {
	AccountID models.AccountID
}

type DeleteBankBridgeConnectionDataFromConnectorID struct {
	ConnectorID models.ConnectorID
}

func (w Workflow) runDeleteBankBridgeConnectionData(
	ctx workflow.Context,
	deleteBankBridgeConnectionData DeleteBankBridgeConnectionData,
) error {
	switch {
	case deleteBankBridgeConnectionData.FromConnectionID != nil:
		// Delete all data related to the connection
		return w.deleteBankBridgeConnectionData(ctx, deleteBankBridgeConnectionData)
	case deleteBankBridgeConnectionData.FromAccountID != nil:
		// Delete only the account and payments related to this account
		return w.deleteBankBridgeConnectionAccountIDData(ctx, deleteBankBridgeConnectionData)
	case deleteBankBridgeConnectionData.FromConnectorID != nil:
		// Delete all data related to the connector
		return w.deleteBankBridgeConnectorIDData(ctx, deleteBankBridgeConnectionData)
	default:
		// Delete all data related to the psu
		return w.deleteBankBridgePSUData(ctx, deleteBankBridgeConnectionData)
	}
}

func (w Workflow) deleteBankBridgeConnectionAccountIDData(
	ctx workflow.Context,
	deleteBankBridgeConnectionData DeleteBankBridgeConnectionData,
) error {
	err := activities.StoragePaymentsDeleteFromAccountID(
		infiniteRetryContext(ctx),
		deleteBankBridgeConnectionData.FromAccountID.AccountID,
	)
	if err != nil {
		return fmt.Errorf("deleting payments from account ID: %w", err)
	}

	err = activities.StorageAccountsDelete(
		infiniteRetryContext(ctx),
		deleteBankBridgeConnectionData.FromAccountID.AccountID,
	)
	if err != nil {
		return fmt.Errorf("deleting account: %w", err)
	}

	return nil
}

func (w Workflow) deleteBankBridgeConnectionData(
	ctx workflow.Context,
	deleteBankBridgeConnectionData DeleteBankBridgeConnectionData,
) error {
	err := w.deleteBankBridgePaymentsFromConnectionID(
		ctx,
		deleteBankBridgeConnectionData.PSUID,
		deleteBankBridgeConnectionData.FromConnectionID.ConnectionID,
	)
	if err != nil {
		return fmt.Errorf("deleting payments: %w", err)
	}

	err = w.deleteBankBridgeAccountsFromConnectionID(
		ctx,
		deleteBankBridgeConnectionData.PSUID,
		deleteBankBridgeConnectionData.FromConnectionID.ConnectionID,
	)
	if err != nil {
		return fmt.Errorf("deleting accounts: %w", err)
	}

	return nil
}

func (w Workflow) deleteBankBridgeConnectorIDData(
	ctx workflow.Context,
	deleteBankBridgeConnectionData DeleteBankBridgeConnectionData,
) error {
	err := w.deleteBankBridgePaymentsFromPSUIDAndConnectorID(
		ctx,
		deleteBankBridgeConnectionData.PSUID,
		deleteBankBridgeConnectionData.FromConnectorID.ConnectorID,
	)
	if err != nil {
		return fmt.Errorf("deleting payments: %w", err)
	}

	err = w.deleteBankBridgeAccountsFromPSUIDAndConnectorID(
		ctx,
		deleteBankBridgeConnectionData.PSUID,
		deleteBankBridgeConnectionData.FromConnectorID.ConnectorID,
	)
	if err != nil {
		return fmt.Errorf("deleting accounts: %w", err)
	}

	return nil
}

func (w Workflow) deleteBankBridgePSUData(
	ctx workflow.Context,
	deleteBankBridgeConnectionData DeleteBankBridgeConnectionData,
) error {
	err := w.deleteBankBridgePaymentsFromPSUID(
		ctx,
		deleteBankBridgeConnectionData.PSUID,
	)
	if err != nil {
		return fmt.Errorf("deleting payments: %w", err)
	}

	err = w.deleteBankBridgeAccountsFromPSUID(
		ctx,
		deleteBankBridgeConnectionData.PSUID,
	)
	if err != nil {
		return fmt.Errorf("deleting accounts: %w", err)
	}

	return nil
}

func (w Workflow) deleteBankBridgePaymentsFromPSUID(
	ctx workflow.Context,
	psuID uuid.UUID,
) error {
	err := activities.StoragePaymentsDeleteFromPSUID(
		infiniteRetryContext(ctx),
		psuID,
	)
	if err != nil {
		return fmt.Errorf("deleting payments: %w", err)
	}

	return nil
}

func (w Workflow) deleteBankBridgePaymentsFromPSUIDAndConnectorID(
	ctx workflow.Context,
	psuID uuid.UUID,
	connectorID models.ConnectorID,
) error {
	err := activities.StoragePaymentsDeleteFromPSUIDAndConnectorID(
		infiniteRetryContext(ctx),
		psuID,
		connectorID,
	)

	if err != nil {
		return fmt.Errorf("deleting payments: %w", err)
	}

	return nil
}

func (w Workflow) deleteBankBridgePaymentsFromConnectionID(
	ctx workflow.Context,
	psuID uuid.UUID,
	connectionID string,
) error {
	err := activities.StoragePaymentsDeleteFromConnectionID(
		infiniteRetryContext(ctx),
		psuID,
		connectionID,
	)

	if err != nil {
		return fmt.Errorf("deleting payments: %w", err)
	}

	return nil
}

func (w Workflow) deleteBankBridgeAccountsFromPSUID(
	ctx workflow.Context,
	psuID uuid.UUID,
) error {
	err := activities.StorageAccountsDeleteFromPSUID(
		infiniteRetryContext(ctx),
		psuID,
	)
	if err != nil {
		return fmt.Errorf("deleting accounts: %w", err)
	}

	return nil
}

func (w Workflow) deleteBankBridgeAccountsFromPSUIDAndConnectorID(
	ctx workflow.Context,
	psuID uuid.UUID,
	connectorID models.ConnectorID,
) error {
	err := activities.StorageAccountsDeleteFromPSUIDAndConnectorID(
		infiniteRetryContext(ctx),
		psuID,
		connectorID,
	)

	if err != nil {
		return fmt.Errorf("deleting accounts: %w", err)
	}

	return nil
}

func (w Workflow) deleteBankBridgeAccountsFromConnectionID(
	ctx workflow.Context,
	psuID uuid.UUID,
	connectionID string,
) error {

	err := activities.StorageAccountsDeleteFromConnectionID(
		infiniteRetryContext(ctx),
		psuID,
		connectionID,
	)

	if err != nil {
		return fmt.Errorf("deleting accounts: %w", err)
	}

	return nil
}

const RunDeleteBankBridgeConnectionData = "DeleteBankBridgeConnectionData"
