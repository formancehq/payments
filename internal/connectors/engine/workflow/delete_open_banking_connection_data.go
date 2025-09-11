package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type DeleteOpenBankingConnectionData struct {
	FromConnectionID *DeleteOpenBankingConnectionDataFromConnectionID
	FromAccountID    *DeleteOpenBankingConnectionDataFromAccountID
	FromConnectorID  *DeleteOpenBankingConnectionDataFromConnectorID
	FromPSUID        *DeleteOpenBankingConnectionDataFromPSUID
}

type DeleteOpenBankingConnectionDataFromConnectionID struct {
	PSUID        uuid.UUID
	ConnectorID  models.ConnectorID
	ConnectionID string
}

type DeleteOpenBankingConnectionDataFromAccountID struct {
	AccountID models.AccountID
}

type DeleteOpenBankingConnectionDataFromConnectorID struct {
	PSUID       uuid.UUID
	ConnectorID models.ConnectorID
}

type DeleteOpenBankingConnectionDataFromPSUID struct {
	PSUID uuid.UUID
}

func (w Workflow) runDeleteOpenBankingConnectionData(
	ctx workflow.Context,
	deleteOpenBankingConnectionData DeleteOpenBankingConnectionData,
) error {
	switch {
	case deleteOpenBankingConnectionData.FromConnectionID != nil:
		// Delete all data related to the connection
		return w.deleteOpenBankingConnectionData(ctx, deleteOpenBankingConnectionData)
	case deleteOpenBankingConnectionData.FromAccountID != nil:
		// Delete only the account and payments related to this account
		return w.deleteOpenBankingConnectionAccountIDData(ctx, deleteOpenBankingConnectionData)
	case deleteOpenBankingConnectionData.FromConnectorID != nil:
		// Delete all data related to the connector
		return w.deleteOpenBankingConnectorIDData(ctx, deleteOpenBankingConnectionData)
	case deleteOpenBankingConnectionData.FromPSUID != nil:
		// Delete all data related to the psu
		return w.deleteOpenBankingPSUData(ctx, deleteOpenBankingConnectionData)
	default:
		return fmt.Errorf("invalid delete open banking connection data")
	}
}

func (w Workflow) deleteOpenBankingConnectionAccountIDData(
	ctx workflow.Context,
	deleteOpenBankingConnectionData DeleteOpenBankingConnectionData,
) error {
	err := activities.StoragePaymentsDeleteFromAccountID(
		infiniteRetryContext(ctx),
		deleteOpenBankingConnectionData.FromAccountID.AccountID,
	)
	if err != nil {
		return fmt.Errorf("deleting payments from account ID: %w", err)
	}

	err = activities.StorageAccountsDelete(
		infiniteRetryContext(ctx),
		deleteOpenBankingConnectionData.FromAccountID.AccountID,
	)
	if err != nil {
		return fmt.Errorf("deleting account: %w", err)
	}

	return nil
}

func (w Workflow) deleteOpenBankingConnectionData(
	ctx workflow.Context,
	deleteOpenBankingConnectionData DeleteOpenBankingConnectionData,
) error {
	err := w.deleteOpenBankingPaymentsFromConnectionID(
		ctx,
		deleteOpenBankingConnectionData.FromConnectionID.PSUID,
		deleteOpenBankingConnectionData.FromConnectionID.ConnectorID,
		deleteOpenBankingConnectionData.FromConnectionID.ConnectionID,
	)
	if err != nil {
		return fmt.Errorf("deleting payments: %w", err)
	}

	err = w.deleteOpenBankingAccountsFromConnectionID(
		ctx,
		deleteOpenBankingConnectionData.FromConnectionID.PSUID,
		deleteOpenBankingConnectionData.FromConnectionID.ConnectorID,
		deleteOpenBankingConnectionData.FromConnectionID.ConnectionID,
	)
	if err != nil {
		return fmt.Errorf("deleting accounts: %w", err)
	}

	return nil
}

func (w Workflow) deleteOpenBankingConnectorIDData(
	ctx workflow.Context,
	deleteOpenBankingConnectionData DeleteOpenBankingConnectionData,
) error {
	err := w.deleteOpenBankingPaymentsFromPSUIDAndConnectorID(
		ctx,
		deleteOpenBankingConnectionData.FromConnectorID.PSUID,
		deleteOpenBankingConnectionData.FromConnectorID.ConnectorID,
	)
	if err != nil {
		return fmt.Errorf("deleting payments: %w", err)
	}

	err = w.deleteOpenBankingAccountsFromPSUIDAndConnectorID(
		ctx,
		deleteOpenBankingConnectionData.FromConnectorID.PSUID,
		deleteOpenBankingConnectionData.FromConnectorID.ConnectorID,
	)
	if err != nil {
		return fmt.Errorf("deleting accounts: %w", err)
	}

	return nil
}

func (w Workflow) deleteOpenBankingPSUData(
	ctx workflow.Context,
	deleteOpenBankingConnectionData DeleteOpenBankingConnectionData,
) error {
	err := w.deleteOpenBankingPaymentsFromPSUID(
		ctx,
		deleteOpenBankingConnectionData.FromPSUID.PSUID,
	)
	if err != nil {
		return fmt.Errorf("deleting payments: %w", err)
	}

	err = w.deleteOpenBankingAccountsFromPSUID(
		ctx,
		deleteOpenBankingConnectionData.FromPSUID.PSUID,
	)
	if err != nil {
		return fmt.Errorf("deleting accounts: %w", err)
	}

	return nil
}

func (w Workflow) deleteOpenBankingPaymentsFromPSUID(
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

func (w Workflow) deleteOpenBankingPaymentsFromPSUIDAndConnectorID(
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

func (w Workflow) deleteOpenBankingPaymentsFromConnectionID(
	ctx workflow.Context,
	psuID uuid.UUID,
	connectorID models.ConnectorID,
	connectionID string,
) error {
	err := activities.StoragePaymentsDeleteFromConnectionID(
		infiniteRetryContext(ctx),
		psuID,
		connectorID,
		connectionID,
	)

	if err != nil {
		return fmt.Errorf("deleting payments: %w", err)
	}

	return nil
}

func (w Workflow) deleteOpenBankingAccountsFromPSUID(
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

func (w Workflow) deleteOpenBankingAccountsFromPSUIDAndConnectorID(
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

func (w Workflow) deleteOpenBankingAccountsFromConnectionID(
	ctx workflow.Context,
	psuID uuid.UUID,
	connectorID models.ConnectorID,
	connectionID string,
) error {

	err := activities.StorageAccountsDeleteFromConnectionID(
		infiniteRetryContext(ctx),
		psuID,
		connectorID,
		connectionID,
	)

	if err != nil {
		return fmt.Errorf("deleting accounts: %w", err)
	}

	return nil
}

const RunDeleteOpenBankingConnectionData = "DeleteOpenBankingConnectionData"
