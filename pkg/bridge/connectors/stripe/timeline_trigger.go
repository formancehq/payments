package stripe

import (
	"context"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/pkg/errors"
	"github.com/stripe/stripe-go/v72"
	"golang.org/x/sync/semaphore"
)

func NewTimelineTrigger(
	logger sharedlogging.Logger,
	ingester Ingester,
	tl *timeline,
) *timelineTrigger {
	return &timelineTrigger{
		logger: logger.WithFields(map[string]interface{}{
			"component": "timeline-trigger",
		}),
		ingester: ingester,
		timeline: tl,
		sem:      semaphore.NewWeighted(1),
	}
}

type timelineTrigger struct {
	logger   sharedlogging.Logger
	ingester Ingester
	timeline *timeline
	sem      *semaphore.Weighted
	cancel   func()
}

func (t *timelineTrigger) Fetch(ctx context.Context) error {
	if t.sem.TryAcquire(1) {
		defer t.sem.Release(1)
		ctx, t.cancel = context.WithCancel(ctx)
		if !t.timeline.State().NoMoreHistory {
			if err := t.fetch(ctx, true); err != nil {
				return err
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := t.fetch(ctx, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *timelineTrigger) Cancel(ctx context.Context) {
	if t.cancel != nil {
		t.cancel()
		t.sem.Acquire(ctx, 1)
		t.sem.Release(1)
	}
}

func (t *timelineTrigger) fetch(ctx context.Context, tail bool) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			hasMore, err := t.triggerPage(ctx, tail)
			if err != nil {
				return errors.Wrap(err, "error triggering tail page")
			}
			if !hasMore {
				return nil
			}
		}
	}
	return nil
}

func (t *timelineTrigger) triggerPage(ctx context.Context, tail bool) (bool, error) {

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
		return false, errors.Wrap(err, "fetching timeline")
	}

	logger.Debug("Ingest batch")
	if len(ret) > 0 {
		err = t.ingester.Ingest(ctx, ret, futureState, tail)
		if err != nil {
			return false, errors.Wrap(err, "ingesting batch")
		}
	}

	commitFn()
	return hasMore, nil
}
