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
) *Runner {
	return &Runner{
		logger:    logger,
		tailToken: make(chan struct{}, 1),
		ingester:  ingester,
		timeline:  tl,
	}
}

type Runner struct {
	name      string
	stopChan  chan chan struct{}
	timeline  *timeline
	config    Config
	tailToken chan struct{}
	logger    sharedlogging.Logger
	ingester  Ingester
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

func (r *Runner) triggerPage(ctx context.Context, tail bool) (bool, error) {

	r.logger.WithFields(map[string]interface{}{
		"tail": tail,
	}).Info("Trigger page")

	ret := make([]stripe.BalanceTransaction, 0)
	method := r.timeline.Head
	if tail {
		method = r.timeline.Tail
	}
	hasMore, futureState, commitFn, err := method(ctx, &ret)
	if err != nil {
		return false, err
	}

	err = r.ingester.Ingest(ctx, ret, futureState, tail)
	if err != nil {
		return false, err
	}

	commitFn()
	return hasMore, nil
}

func (r *Runner) Run(ctx context.Context) error {

	r.logger.WithFields(map[string]interface{}{
		"config": r.config,
		"state":  r.timeline.State(),
	}).Info("Starting runner")

	r.stopChan = make(chan chan struct{}, 1)

	r.triggerPage(ctx, false)
	r.tailToken <- struct{}{}

	var timer *time.Timer
	resetTimer := func() {
		timer = time.NewTimer(r.config.PollingPeriod)
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
						case <-time.After(r.config.PollingPeriod):
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
