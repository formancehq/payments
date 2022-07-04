package stripe

import (
	"context"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge/ingestion"
	"github.com/numary/payments/pkg/bridge/task"
	"github.com/pkg/errors"
	"github.com/stripe/stripe-go/v72"
)

func ingest(
	ctx context.Context,
	logger sharedlogging.Logger,
	scheduler task.Scheduler[TaskDescriptor],
	ingester ingestion.Ingester,
	bts []*stripe.BalanceTransaction,
	commitState TimelineState,
	tail bool) error {
	connectedAccounts := make([]string, 0)

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
		if bt.Type == stripe.BalanceTransactionTypeTransfer {
			connectedAccounts = append(connectedAccounts, bt.Source.Transfer.Destination.ID)
		}
	}
	logger.WithFields(map[string]interface{}{
		"state": commitState,
	}).Debugf("updating state")

	for _, connectedAccount := range connectedAccounts {
		err := scheduler.Schedule(TaskDescriptor{
			Account: connectedAccount,
		}, true)
		if err != nil && err != task.ErrAlreadyScheduled {
			return errors.Wrap(err, "scheduling connected account")
		}
	}

	err := ingester.Ingest(ctx, batch, commitState)
	if err != nil {
		return errors.Wrap(err, "ingesting batch")
	}

	return nil
}

func MainTask(client Client, config Config) func(ctx context.Context, logger sharedlogging.Logger, resolver task.StateResolver,
	scheduler task.Scheduler[TaskDescriptor], ingester ingestion.Ingester) error {
	return func(ctx context.Context, logger sharedlogging.Logger, resolver task.StateResolver,
		scheduler task.Scheduler[TaskDescriptor], ingester ingestion.Ingester) error {
		runner := NewRunner(
			logger,
			NewTimelineTrigger(
				logger,
				IngesterFn(func(ctx context.Context, batch []*stripe.BalanceTransaction, commitState TimelineState, tail bool) error {
					return ingest(ctx, logger, scheduler, ingester, batch, commitState, tail)
				}),
				NewTimeline(client, config.TimelineConfig, task.MustResolveTo(ctx, resolver, TimelineState{})),
			),
			config.PollingPeriod,
		)
		return runner.Run(ctx)
	}
}
