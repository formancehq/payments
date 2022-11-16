package bankingcircle

import (
	"context"

	"github.com/numary/payments/internal/pkg/ingestion"
	"github.com/numary/payments/internal/pkg/payments"
	"github.com/numary/payments/internal/pkg/task"

	"github.com/numary/go-libs/sharedlogging"
)

func taskFetchPayments(logger sharedlogging.Logger, client *client) task.Task {
	return func(
		ctx context.Context,
		scheduler task.Scheduler[TaskDescriptor],
		ingester ingestion.Ingester,
	) error {
		paymentsList, err := client.getAllPayments()
		if err != nil {
			return err
		}

		batch := ingestion.Batch{}

		for _, paymentEl := range paymentsList {
			logger.Info(paymentEl)

			batchElement := ingestion.BatchElement{
				Referenced: payments.Referenced{
					Reference: paymentEl.TransactionReference,
					Type:      matchPaymentType(paymentEl.Classification),
				},
				Payment: &payments.Data{
					Status:        matchPaymentStatus(paymentEl.Status),
					Scheme:        payments.SchemeOther,
					InitialAmount: int64(paymentEl.Transfer.Amount.Amount * 100),
					Asset:         paymentEl.Transfer.Amount.Currency + "/2",
					Raw:           paymentEl,
				},
			}

			batch = append(batch, batchElement)
		}

		return ingester.Ingest(ctx, batch, struct{}{})
	}
}

func matchPaymentStatus(paymentStatus string) payments.Status {
	switch paymentStatus {
	case "Processed":
		return payments.StatusSucceeded
	// On MissingFunding - the payment is still in progress.
	// If there will be funds available within 10 days - the payment will be processed.
	// Otherwise - it will be cancelled.
	case "PendingProcessing", "MissingFunding":
		return payments.StatusPending
	case "Rejected", "Cancelled", "Reversed", "Returned":
		return payments.StatusFailed
	}

	return payments.TypeOther
}

func matchPaymentType(paymentType string) string {
	switch paymentType {
	case "Incoming":
		return payments.TypePayIn
	case "Outgoing":
		return payments.TypePayout
	}

	return payments.TypeOther
}
