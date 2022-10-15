package wise

import (
	"context"
	"fmt"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/internal/pkg/ingestion"
	"github.com/numary/payments/internal/pkg/payments"
	"github.com/numary/payments/internal/pkg/task"
)

func taskFetchTransfers(logger sharedlogging.Logger, client *client, profileID uint64) task.Task {
	return func(
		ctx context.Context,
		scheduler task.Scheduler[TaskDescriptor],
		ingester ingestion.Ingester,
	) error {
		transfers, err := client.getTransfers(&profile{
			ID: profileID,
		})
		if err != nil {
			return err
		}

		batch := ingestion.Batch{}

		for _, transfer := range transfers {
			logger.Info(transfer)

			batchElement := ingestion.BatchElement{
				Referenced: payments.Referenced{
					Reference: fmt.Sprintf("%d", transfer.ID),
					Type:      payments.TypeTransfer,
				},
				Payment: &payments.Data{
					Status:        matchTransferStatus(transfer.Status),
					Scheme:        payments.SchemeOther,
					InitialAmount: int64(transfer.TargetValue * 100),
					Asset:         fmt.Sprintf("%s/2", transfer.TargetCurrency),
					Raw:           transfer,
				},
			}

			batch = append(batch, batchElement)
		}

		return ingester.Ingest(ctx, batch, struct{}{})
	}
}

func matchTransferStatus(status string) payments.Status {
	switch status {
	case "incoming_payment_waiting", "processing":
		return payments.StatusPending
	case "funds_converted", "outgoing_payment_sent":
		return payments.StatusSucceeded
	case "bounced_back", "funds_refunded":
		return payments.StatusFailed
	case "canceled":
		return payments.StatusCancelled
	}

	return payments.StatusOther
}
