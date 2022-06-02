package stripe

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/stripe/stripe-go/v72"
)

func NewTimelineTrigger(
	logger sharedlogging.Logger,
	ingester Ingester,
	tl *timeline,
) *TimelineTrigger {
	return &TimelineTrigger{
		logger:   logger,
		ingester: ingester,
		timeline: tl,
	}
}

type TimelineTrigger struct {
	logger   sharedlogging.Logger
	ingester Ingester
	timeline *timeline
}

func (t *TimelineTrigger) Fetch(ctx context.Context) {
	if !t.timeline.State().NoMoreHistory {
		t.fetch(ctx, true)
	}
	t.fetch(ctx, false)
}

func (t *TimelineTrigger) fetch(ctx context.Context, tail bool) {
	for {
		hasMore, err := t.triggerPage(ctx, tail)
		if err != nil {
			t.logger.Errorf("error triggering tail page: %s", err)
			continue
		}
		if !hasMore {
			return
		}
	}
}

func (t *TimelineTrigger) triggerPage(ctx context.Context, tail bool) (bool, error) {

	logger := t.logger.WithFields(map[string]interface{}{
		"tail": tail,
	})
	logger.Debugf("Trigger page")

	ret := make([]*stripe.BalanceTransaction, 0)
	method := t.timeline.Head
	if tail {
		method = t.timeline.Tail
	}

	hasMore, futureState, commitFn, err := method(ctx, &ret)
	if err != nil {
		return false, err
	}

	logger.Debug("Ingest batch")
	err = t.ingester.Ingest(ctx, ret, futureState, tail)
	if err != nil {
		return false, err
	}

	commitFn()
	return hasMore, nil
}
