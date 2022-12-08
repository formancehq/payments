package currencycloud

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/app/connectors/currencycloud/client"

	"github.com/formancehq/payments/internal/app/ingestion"
	"github.com/formancehq/payments/internal/app/payments"
	"github.com/formancehq/payments/internal/app/task"

	"github.com/formancehq/go-libs/sharedlogging"
)

func taskFetchTransactions(logger sharedlogging.Logger, client *client.Client, config Config) task.Task {
	return func(
		ctx context.Context,
		ingester ingestion.Ingester,
	) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(config.PollingPeriod.Duration()):
				if err := ingestTransactions(ctx, logger, client, ingester); err != nil {
					return err
				}
			}
		}
	}
}

func ingestTransactions(ctx context.Context, logger sharedlogging.Logger,
	client *client.Client, ingester ingestion.Ingester,
) error {
	page := 1

	for {
		if page < 0 {
			break
		}

		logger.Info("Fetching transactions")

		transactions, nextPage, err := client.GetTransactions(ctx, page)
		if err != nil {
			return err
		}

		page = nextPage

		batch := ingestion.PaymentBatch{}

		for _, transaction := range transactions {
			logger.Info(transaction)

			var amount float64

			amount, err = strconv.ParseFloat(transaction.Amount, 64)
			if err != nil {
				return fmt.Errorf("failed to parse amount: %w", err)
			}

			batchElement := ingestion.PaymentBatchElement{
				Referenced: payments.Referenced{
					Reference: transaction.ID,
					Type:      matchTransactionType(transaction.Type),
				},
				Payment: &payments.Data{
					Status:        matchTransactionStatus(transaction.Status),
					Scheme:        payments.SchemeOther,
					InitialAmount: int64(amount * 100),
					Asset:         fmt.Sprintf("%s/2", transaction.Currency),
					Raw:           transaction,
				},
			}

			batch = append(batch, batchElement)
		}

		err = ingester.IngestPayments(ctx, batch, struct{}{})
		if err != nil {
			return err
		}
	}

	return nil
}

func matchTransactionType(transactionType string) string {
	switch transactionType {
	case "credit":
		return payments.TypePayout
	case "debit":
		return payments.TypePayIn
	}

	return payments.TypeOther
}

func matchTransactionStatus(transactionStatus string) payments.Status {
	switch transactionStatus {
	case "completed":
		return payments.StatusSucceeded
	case "pending":
		return payments.StatusPending
	case "deleted":
		return payments.StatusFailed
	}

	return payments.TypeOther
}
