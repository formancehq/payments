package bankingcircle

import (
	"context"

	"github.com/numary/go-libs/sharedlogging"
	payments "github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge/ingestion"
	"github.com/numary/payments/pkg/bridge/task"
)

func taskFetchPayments(logger sharedlogging.Logger, client *client) task.Task {
	return func(
		ctx context.Context,
		scheduler task.Scheduler[TaskDefinition],
		ingester ingestion.Ingester,
	) error {
		paymentsList, err := client.getAllPayments()
		if err != nil {
			return err
		}

		batch := ingestion.Batch{}

		for _, payment := range paymentsList {
			logger.Info(payment)

			batchElement := ingestion.BatchElement{
				Referenced: payments.Referenced{
					Reference: payment.TransactionReference,
					Type:      matchPaymentType(payment.Classification),
				},
				Payment: &payments.Data{
					Status:        matchPaymentStatus(payment.Status),
					Scheme:        payments.SchemeOther,
					InitialAmount: int64(payment.Transfer.Amount.Amount * 100),
					Asset:         payment.Transfer.Amount.Currency + "/2",
					Raw:           payment,
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
