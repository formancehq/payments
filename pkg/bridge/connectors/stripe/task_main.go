package stripe

import (
	"context"
	"github.com/numary/payments/pkg/bridge/ingestion"
	"github.com/numary/payments/pkg/bridge/task"
	"github.com/pkg/errors"
	"github.com/stripe/stripe-go/v72"
)

type mainTask struct {
	runner *Runner
	config Config
	client Client
}

func (p *mainTask) Cancel(ctx context.Context) error {
	return p.runner.Stop(ctx)
}

func (p *mainTask) Ingest(ctx task.Context[TaskDescriptor, TimelineState], bts []*stripe.BalanceTransaction, commitState TimelineState, tail bool) error {
	connectedAccounts := make([]string, 0)

	batch := ingestion.Batch{}
	for _, bt := range bts {
		batchElement, handled := CreateBatchElement(bt, !tail)
		if !handled {
			ctx.Logger().Debugf("Balance transaction type not handled: %s", bt.Type)
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
	ctx.Logger().WithFields(map[string]interface{}{
		"state": commitState,
	}).Debugf("updating state")

	for _, connectedAccount := range connectedAccounts {
		err := ctx.Scheduler().Schedule(TaskDescriptor{
			Account: connectedAccount,
		})
		if err != nil && err != task.ErrAlreadyScheduled {
			return errors.Wrap(err, "scheduling connected account")
		}
	}

	err := ctx.Ingester().Ingest(ctx.Context(), batch, commitState)
	if err != nil {
		return errors.Wrap(err, "ingesting batch")
	}

	return nil
}

func (p *mainTask) Run(taskCtx task.Context[TaskDescriptor, TimelineState]) error {

	p.runner = NewRunner(
		taskCtx.Logger(),
		NewTimelineTrigger(
			taskCtx.Logger(),
			IngesterFn(func(ctx context.Context, batch []*stripe.BalanceTransaction, commitState TimelineState, tail bool) error {
				return p.Ingest(taskCtx.WithContext(ctx), batch, commitState, tail)
			}),
			NewTimeline(p.client, p.config.TimelineConfig, taskCtx.State()),
		),
		p.config.PollingPeriod,
	)

	return p.runner.Run(taskCtx.Context())
}

func (p *mainTask) Name() string {
	return "main"
}

var _ task.Task[TaskDescriptor, TimelineState] = &mainTask{}
