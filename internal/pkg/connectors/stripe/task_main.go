package stripe

import (
	"context"
	"net/http"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/internal/pkg/ingestion"
	"github.com/numary/payments/internal/pkg/task"
	"github.com/numary/payments/internal/pkg/writeonly"
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
	tail bool,
) error {
	err := ingestBatch(ctx, logger, ingester, bts, commitState, tail)
	if err != nil {
		return err
	}

	connectedAccounts := make([]string, 0)

	for _, bt := range bts {
		if bt.Type == stripe.BalanceTransactionTypeTransfer {
			connectedAccounts = append(connectedAccounts, bt.Source.Transfer.Destination.ID)
		}
	}

	for _, connectedAccount := range connectedAccounts {
		err = scheduler.Schedule(TaskDescriptor{
			Account: connectedAccount,
		}, true)
		if err != nil && !errors.Is(err, task.ErrAlreadyScheduled) {
			return errors.Wrap(err, "scheduling connected account")
		}
	}

	return nil
}

func MainTask(config Config) func(ctx context.Context, logger sharedlogging.Logger, resolver task.StateResolver,
	scheduler task.Scheduler[TaskDescriptor], ingester ingestion.Ingester, storage writeonly.Storage) error {
	return func(ctx context.Context, logger sharedlogging.Logger, resolver task.StateResolver,
		scheduler task.Scheduler[TaskDescriptor], ingester ingestion.Ingester, storage writeonly.Storage,
	) error {
		runner := NewRunner(
			logger,
			NewTimelineTrigger(
				logger,
				IngesterFn(func(ctx context.Context, batch []*stripe.BalanceTransaction,
					commitState TimelineState, tail bool,
				) error {
					return ingest(ctx, logger, scheduler, ingester, batch, commitState, tail)
				}),
				NewTimeline(NewDefaultClient(http.DefaultClient, config.APIKey, storage),
					config.TimelineConfig, task.MustResolveTo(ctx, resolver, TimelineState{})),
			),
			config.PollingPeriod.Duration,
		)

		return runner.Run(ctx)
	}
}