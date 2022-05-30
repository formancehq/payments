package stripe

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/pkg/errors"
	"github.com/stripe/stripe-go/v72"
	"time"
)

func NewRunner(
	logger sharedlogging.Logger,
	ingester Ingester,
	tl *timeline,
	pollingPeriod time.Duration,
) *Runner {
	return &Runner{
		logger:        logger,
		tailToken:     make(chan struct{}, 1),
		ingester:      ingester,
		timeline:      tl,
		pollingPeriod: pollingPeriod,
	}
}

type Runner struct {
	name          string
	stopChan      chan chan struct{}
	timeline      *timeline
	tailToken     chan struct{}
	logger        sharedlogging.Logger
	ingester      Ingester
	pollingPeriod time.Duration
}

func (r *Runner) Stop(ctx context.Context) error {
	ch := make(chan struct{})
	select {
	case r.stopChan <- ch:
		select {
		case <-ch:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-ctx.Done():
		return ctx.Err()
	default:
		return errors.New("already closed")
	}
}

func (r *Runner) IsTailing() bool {
	return r.tailToken != nil
}

func (r *Runner) triggerPage(ctx context.Context, tail bool) (bool, error) {

	logger := r.logger.WithFields(map[string]interface{}{
		"tail": tail,
	})
	logger.Info("Trigger page")

	ret := make([]*stripe.BalanceTransaction, 0)
	method := r.timeline.Head
	if tail {
		method = r.timeline.Tail
	}

	hasMore, futureState, commitFn, err := method(ctx, &ret)
	if err != nil {
		return false, err
	}

	logger.Infof("ingest")
	err = r.ingester.Ingest(ctx, ret, futureState, tail)
	if err != nil {
		return false, err
	}

	commitFn()
	return hasMore, nil
}

func (r *Runner) Run(ctx context.Context) error {

	r.logger.WithFields(map[string]interface{}{
		"polling-period": r.pollingPeriod,
	}).Info("Starting runner")

	r.stopChan = make(chan chan struct{}, 1)

	r.triggerPage(ctx, false)
	r.tailToken <- struct{}{}

	var timer *time.Timer
	resetTimer := func() {
		timer = time.NewTimer(r.pollingPeriod)
	}
	resetTimer()

	var closeChannel chan struct{}
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case closeChannel = <-r.stopChan:
			cancel()
		case <-ctx.Done():
		}
	}()

l:
	for {
		select {
		case <-ctx.Done():
			if closeChannel != nil {
				close(closeChannel)
			}
			return nil
		case <-timer.C:
			hasMore := true
			var err error
			for hasMore {
				hasMore, err = r.triggerPage(ctx, false)
				if err != nil {
					r.logger.Errorf("Error fetching page: %s", err)
					break
				}
				select {
				case <-ctx.Done():
					continue l
				default:
					// Nothing to do
				}
			}
			resetTimer()
		default:
			select {
			case <-ctx.Done():
			case <-r.tailToken:
				hasMore, err := r.triggerPage(ctx, true)
				if err != nil {
					r.logger.Errorf("Error fetching page: %s", err)
					go func() {
						select {
						case <-time.After(r.pollingPeriod):
						case <-ctx.Done():
							return
						}
						select {
						case r.tailToken <- struct{}{}:
						case <-ctx.Done():
						}
					}()
					break
				}
				if hasMore {
					r.tailToken <- struct{}{}
				} else {
					r.tailToken = nil
				}
			default:
				// Nothing to do
			}
		}
	}
}
