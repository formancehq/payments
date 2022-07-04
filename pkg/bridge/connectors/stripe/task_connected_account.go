package stripe

import (
	"context"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge/ingestion"
	"github.com/numary/payments/pkg/bridge/task"
	"github.com/stripe/stripe-go/v72"
)

func ingestBatch(ctx context.Context, logger sharedlogging.Logger, ingester ingestion.Ingester, bts []*stripe.BalanceTransaction, commitState TimelineState, tail bool) error {
	batch := ingestion.Batch{}
	for _, bt := range bts {
		batchElement, handled := CreateBatchElement(bt, !tail)
		if !handled {
			logger.Debugf("Balance transaction type not handled: %s", bt.Type)
			continue
		}
		if batchElement.Adjustment == nil && batchElement.Payment == nil {
			continue
		}
		batch = append(batch, batchElement)
	}
	logger.WithFields(map[string]interface{}{
		"state": commitState,
	}).Debugf("updating state")

	err := ingester.Ingest(ctx, batch, commitState)
	if err != nil {
		return err
	}

	return nil
}

func ConnectedAccountTask(client Client, config Config, account string) func(ctx context.Context, logger sharedlogging.Logger,
	ingester ingestion.Ingester, resolver task.StateResolver) error {
	return func(ctx context.Context, logger sharedlogging.Logger, ingester ingestion.Ingester, resolver task.StateResolver) error {
		logger.Infof("Create new trigger")

		trigger := NewTimelineTrigger(
			logger,
			IngesterFn(func(ctx context.Context, bts []*stripe.BalanceTransaction, commitState TimelineState, tail bool) error {
				return ingestBatch(ctx, logger, ingester, bts, commitState, tail)
			}),
			NewTimeline(client.ForAccount(account), config.TimelineConfig, task.MustResolveTo(ctx, resolver, TimelineState{})),
		)
		return trigger.Fetch(ctx)
	}
}
