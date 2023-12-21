package currencycloud

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"

	"github.com/formancehq/payments/cmd/connectors/internal/connectors/currency"
	"github.com/formancehq/payments/cmd/connectors/internal/connectors/currencycloud/client"
	"github.com/formancehq/payments/cmd/connectors/internal/ingestion"
	"github.com/formancehq/payments/cmd/connectors/internal/task"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/stack/libs/go-libs/logging"
)

func taskFetchTransactions(logger logging.Logger, client *client.Client, config Config) task.Task {
	return func(
		ctx context.Context,
		connectorID models.ConnectorID,
		ingester ingestion.Ingester,
	) error {
		return ingestTransactions(ctx, logger, connectorID, client, ingester)
	}
}

func ingestTransactions(
	ctx context.Context,
	logger logging.Logger,
	connectorID models.ConnectorID,
	client *client.Client,
	ingester ingestion.Ingester,
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

			precision, ok := supportedCurrenciesWithDecimal[transaction.Currency]
			if !ok {
				logger.Errorf("currency %s is not supported", transaction.Currency)
				continue
			}

			var amount big.Float
			_, ok = amount.SetString(transaction.Amount)
			if !ok {
				return fmt.Errorf("failed to parse amount %s", transaction.Amount)
			}
			var amountInt big.Int
			amount.Mul(&amount, big.NewFloat(math.Pow(10, float64(precision)))).Int(&amountInt)

			var rawData json.RawMessage

			rawData, err = json.Marshal(transaction)
			if err != nil {
				return fmt.Errorf("failed to marshal transaction: %w", err)
			}

			paymentType := matchTransactionType(transaction.Type)

			batchElement := ingestion.PaymentBatchElement{
				Payment: &models.Payment{
					ID: models.PaymentID{
						PaymentReference: models.PaymentReference{
							Reference: transaction.ID,
							Type:      paymentType,
						},
						ConnectorID: connectorID,
					},
					Reference:     transaction.ID,
					Type:          paymentType,
					ConnectorID:   connectorID,
					Status:        matchTransactionStatus(transaction.Status),
					Scheme:        models.PaymentSchemeOther,
					Amount:        &amountInt,
					InitialAmount: &amountInt,
					Asset:         currency.FormatAsset(supportedCurrenciesWithDecimal, transaction.Currency),
					RawData:       rawData,
				},
			}

			switch paymentType {
			case models.PaymentTypePayOut:
				batchElement.Payment.SourceAccountID = &models.AccountID{
					Reference:   transaction.AccountID,
					ConnectorID: connectorID,
				}
			default:
				batchElement.Payment.DestinationAccountID = &models.AccountID{
					Reference:   transaction.AccountID,
					ConnectorID: connectorID,
				}
			}

			batch = append(batch, batchElement)
		}

		err = ingester.IngestPayments(ctx, connectorID, batch, struct{}{})
		if err != nil {
			return err
		}
	}

	return nil
}

func matchTransactionType(transactionType string) models.PaymentType {
	switch transactionType {
	case "credit":
		return models.PaymentTypePayOut
	case "debit":
		return models.PaymentTypePayIn
	}

	return models.PaymentTypeOther
}

func matchTransactionStatus(transactionStatus string) models.PaymentStatus {
	switch transactionStatus {
	case "completed":
		return models.PaymentStatusSucceeded
	case "pending":
		return models.PaymentStatusPending
	case "deleted":
		return models.PaymentStatusFailed
	}

	return models.PaymentStatusOther
}