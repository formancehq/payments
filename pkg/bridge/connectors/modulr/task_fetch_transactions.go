package modulr

import (
	"context"
	"fmt"
	"strings"

	"github.com/numary/go-libs/sharedlogging"
	payments "github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge/connectors/modulr/client"
	"github.com/numary/payments/pkg/bridge/ingestion"
	"github.com/numary/payments/pkg/bridge/task"
)

func taskFetchTransactions(logger sharedlogging.Logger, client *client.Client, accountID string) task.Task {
	return func(
		ctx context.Context,
		ingester ingestion.Ingester,
	) error {
		logger.Info("Fetching transactions for account", accountID)

		transactions, err := client.GetTransactions(accountID)
		if err != nil {
			return err
		}

		batch := ingestion.Batch{}

		for _, transaction := range transactions {
			logger.Info(transaction)

			batchElement := ingestion.BatchElement{
				Referenced: payments.Referenced{
					Reference: transaction.ID,
					Type:      matchTransactionType(transaction.Type),
				},
				Payment: &payments.Data{
					// API only retrieves successful payments
					Status:        payments.StatusSucceeded,
					Scheme:        payments.SchemeOther,
					InitialAmount: int64(transaction.Amount * 100),
					Asset:         fmt.Sprintf("%s/2", transaction.Account.Currency),
					Raw:           transaction,
				},
			}

			batch = append(batch, batchElement)
		}

		return ingester.Ingest(ctx, batch, struct{}{})
	}
}

func matchTransactionType(transactionType string) string {
	if transactionType == "PI_REV" ||
		transactionType == "PO_REV" ||
		transactionType == "ADHOC" ||
		transactionType == "INT_INTERC" {
		return payments.TypeOther
	}

	if strings.HasPrefix(transactionType, "PI_") {
		return payments.TypePayIn
	}

	if strings.HasPrefix(transactionType, "PO_") {
		return payments.TypePayout
	}

	return payments.TypeOther
}
