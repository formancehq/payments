package wise

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/pkg/ingestion"
	"github.com/formancehq/payments/internal/pkg/payments"
	"github.com/formancehq/payments/internal/pkg/task"
	"github.com/numary/go-libs/sharedlogging"
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

		var (
			accountBatch ingestion.AccountBatch
			paymentBatch ingestion.PaymentBatch
		)

		for _, transfer := range transfers {
			logger.Info(transfer)

			batchElement := ingestion.PaymentBatchElement{
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

			if transfer.SourceAccount != 0 {
				ref := fmt.Sprintf("%d", transfer.SourceAccount)

				accountBatch = append(accountBatch,
					ingestion.AccountBatchElement{
						Reference: ref,
						Type:      payments.AccountTypeSource,
					},
				)

				batchElement.Referenced.Accounts = append(batchElement.Referenced.Accounts, ref)
			}

			if transfer.TargetAccount != 0 {
				ref := fmt.Sprintf("%d", transfer.TargetAccount)

				accountBatch = append(accountBatch,
					ingestion.AccountBatchElement{
						Reference: ref,
						Type:      payments.AccountTypeTarget,
					},
				)

				batchElement.Referenced.Accounts = append(batchElement.Referenced.Accounts, ref)
			}

			paymentBatch = append(paymentBatch, batchElement)
		}

		err = ingester.IngestAccounts(ctx, accountBatch)
		if err != nil {
			return err
		}

		return ingester.IngestPayments(ctx, paymentBatch, struct{}{})
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
	case "cancelled":
		return payments.StatusCancelled
	}

	return payments.StatusOther
}
