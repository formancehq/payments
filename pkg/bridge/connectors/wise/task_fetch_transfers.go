package wise

import (
	"context"
	"fmt"

	"github.com/numary/go-libs/sharedlogging"
	payments "github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge/ingestion"
	"github.com/numary/payments/pkg/bridge/task"
)

func taskFetchTransfers(logger sharedlogging.Logger, config Config, profileID uint64) task.Task {
	return func(
		ctx context.Context,
		scheduler task.Scheduler[TaskDefinition],
		ingester ingestion.Ingester,
	) error {
		client := newClient(config.APIKey)

		transfers, err := client.getTransfers(&profile{
			ID: profileID,
		})

		if err != nil {
			return err
		}

		batch := ingestion.Batch{}

		for _, transfer := range transfers {
			logger.Info(transfer)
			batch = append(batch, ingestion.BatchElement{
				Referenced: payments.Referenced{
					Reference: fmt.Sprintf("%d", transfer.ID),
					Type:      "transfer",
				},
				Payment: &payments.Data{
					Status:        payments.StatusSucceeded,
					Scheme:        payments.SchemeOther,
					InitialAmount: int64(transfer.TargetValue * 100),
					Asset:         fmt.Sprintf("%s/2", transfer.TargetCurrency),
					Raw:           transfer,
				},
			})
		}

		return ingester.Ingest(ctx, batch, struct{}{})
	}
}
