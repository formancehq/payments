package workflow

import (
	"fmt"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type DeleteOpenBankingConnectionData struct {
	PSUID uuid.UUID

	FromConnectionID *DeleteOpenBankingConnectionDataFromConnectionID
	FromAccountID    *DeleteOpenBankingConnectionDataFromAccountID
	FromConnectorID  *DeleteOpenBankingConnectionDataFromConnectorID
}

type DeleteOpenBankingConnectionDataFromConnectionID struct {
	ConnectionID string
}

type DeleteOpenBankingConnectionDataFromAccountID struct {
	AccountID models.AccountID
}

type DeleteOpenBankingConnectionDataFromConnectorID struct {
	ConnectorID models.ConnectorID
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
	default:
		// Delete all data related to the psu
		return w.deleteOpenBankingPSUData(ctx, deleteOpenBankingConnectionData)
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
	err := w.deleteOpenBankingPayments(
		ctx,
		map[string]string{
			models.ObjectConnectionIDMetadataKey: deleteOpenBankingConnectionData.FromConnectionID.ConnectionID,
		},
	)
	if err != nil {
		return fmt.Errorf("deleting payments: %w", err)
	}

	err = w.deleteOpenBankingAccounts(
		ctx,
		map[string]string{
			models.ObjectConnectionIDMetadataKey: deleteOpenBankingConnectionData.FromConnectionID.ConnectionID,
		},
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
	err := w.deleteOpenBankingPayments(
		ctx,
		map[string]string{
			models.ObjectPSUIDMetadataKey: deleteOpenBankingConnectionData.PSUID.String(),
			"connector_id":                deleteOpenBankingConnectionData.FromConnectorID.ConnectorID.String(),
		},
	)
	if err != nil {
		return fmt.Errorf("deleting payments: %w", err)
	}

	err = w.deleteOpenBankingAccounts(
		ctx,
		map[string]string{
			models.ObjectPSUIDMetadataKey: deleteOpenBankingConnectionData.PSUID.String(),
			"connector_id":                deleteOpenBankingConnectionData.FromConnectorID.ConnectorID.String(),
		},
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
	err := w.deleteOpenBankingPayments(
		ctx,
		map[string]string{
			models.ObjectPSUIDMetadataKey: deleteOpenBankingConnectionData.PSUID.String(),
		},
	)
	if err != nil {
		return fmt.Errorf("deleting payments: %w", err)
	}

	err = w.deleteOpenBankingAccounts(
		ctx,
		map[string]string{
			models.ObjectPSUIDMetadataKey: deleteOpenBankingConnectionData.PSUID.String(),
		},
	)
	if err != nil {
		return fmt.Errorf("deleting accounts: %w", err)
	}

	return nil
}

func (w Workflow) deleteOpenBankingPayments(
	ctx workflow.Context,
	filteredMetadata map[string]string,
) error {
	var q query.Builder
	matches := []query.Builder{}
	for key, value := range filteredMetadata {
		matches = append(matches, query.Match(fmt.Sprintf("metadata[%s]", key), value))
	}
	if len(matches) > 1 {
		q = query.And(matches...)
	} else {
		q = matches[0]
	}

	query := storage.NewListPaymentsQuery(
		bunpaginate.NewPaginatedQueryOptions(storage.PaymentQuery{}).
			WithPageSize(50).
			WithQueryBuilder(q),
	)

	for {
		cursor, err := activities.StoragePaymentsList(
			infiniteRetryContext(ctx),
			query,
		)
		if err != nil {
			return err
		}

		wg := workflow.NewWaitGroup(ctx)

		for _, payment := range cursor.Data {
			payment := payment
			wg.Add(1)
			workflow.Go(ctx, func(ctx workflow.Context) {
				defer wg.Done()

				if err := activities.StoragePaymentsDelete(
					infiniteRetryContext(ctx),
					payment.ID,
				); err != nil {
					workflow.GetLogger(ctx).Error("failed to delete payment", "payment_id", payment.ID, "error", err)
				}
			})
		}

		wg.Wait(ctx)

		if !cursor.HasMore {
			break
		}

		err = bunpaginate.UnmarshalCursor(cursor.Next, &query)
		if err != nil {
			return err
		}
	}

	return nil
}

func (w Workflow) deleteOpenBankingAccounts(
	ctx workflow.Context,
	filteredMetadata map[string]string,
) error {
	var q query.Builder
	matches := []query.Builder{}
	for key, value := range filteredMetadata {
		matches = append(matches, query.Match(fmt.Sprintf("metadata[%s]", key), value))
	}
	if len(matches) > 1 {
		q = query.And(matches...)
	} else {
		q = matches[0]
	}

	query := storage.NewListAccountsQuery(
		bunpaginate.NewPaginatedQueryOptions(storage.AccountQuery{}).
			WithPageSize(50).
			WithQueryBuilder(q),
	)

	for {
		cursor, err := activities.StorageAccountsList(
			infiniteRetryContext(ctx),
			query,
		)
		if err != nil {
			return err
		}

		wg := workflow.NewWaitGroup(ctx)

		for _, account := range cursor.Data {
			account := account
			wg.Add(1)
			workflow.Go(ctx, func(ctx workflow.Context) {
				defer wg.Done()

				if err := activities.StorageAccountsDelete(
					infiniteRetryContext(ctx),
					account.ID,
				); err != nil {
					workflow.GetLogger(ctx).Error("failed to delete account", "account_id", account.ID, "error", err)
				}
			})
		}

		wg.Wait(ctx)

		if !cursor.HasMore {
			break
		}

		err = bunpaginate.UnmarshalCursor(cursor.Next, &query)
		if err != nil {
			return err
		}
	}

	return nil
}

const RunDeleteOpenBankingConnectionData = "DeleteOpenBankingConnectionData"
