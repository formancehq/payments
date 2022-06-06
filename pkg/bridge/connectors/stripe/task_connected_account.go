package stripe

import (
	"context"
	"github.com/numary/payments/pkg/bridge/ingestion"
	"github.com/numary/payments/pkg/bridge/task"
	"github.com/stripe/stripe-go/v72"
)

type connectedAccountTask struct {
	account string
	client  Client
	config  TimelineConfig
	trigger *timelineTrigger
}

func (p *connectedAccountTask) Name() string {
	return "connected-account:" + p.account
}

func (p *connectedAccountTask) Run(taskContext task.Context[TaskDescriptor, TimelineState]) error {
	taskContext.Logger().Infof("Create new trigger")

	p.trigger = NewTimelineTrigger(
		taskContext.Logger(),
		IngesterFn(func(ctx context.Context, bts []*stripe.BalanceTransaction, commitState TimelineState, tail bool) error {

			batch := ingestion.Batch{}
			for _, bt := range bts {
				batchElement, handled := CreateBatchElement(bt, !tail)
				if !handled {
					taskContext.Logger().Debugf("Balance transaction type not handled: %s", bt.Type)
					continue
				}
				if batchElement.Adjustment == nil && batchElement.Payment == nil {
					continue
				}
				batch = append(batch, batchElement)
			}
			taskContext.Logger().WithFields(map[string]interface{}{
				"state": commitState,
			}).Debugf("updating state")

			err := taskContext.Ingester().Ingest(ctx, batch, commitState)
			if err != nil {
				return err
			}

			return nil
		}),
		NewTimeline(p.client.ForAccount(p.account), p.config, taskContext.State()),
	)
	return p.trigger.Fetch(taskContext.Context())
}

func (p *connectedAccountTask) Cancel(ctx context.Context) error {
	p.trigger.Cancel(ctx)
	return nil
}

var _ task.Task[TaskDescriptor, TimelineState] = &connectedAccountTask{}
