package stripe

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge"
	"github.com/pkg/errors"
	"github.com/stripe/stripe-go/v72"
	"time"
)

func NewRunner(name string, logObjectStorage bridge.LogObjectStorage, logger sharedlogging.Logger, ingester bridge.Ingester[Config, State, *Connector], config Config, state State) *Runner {
	return &Runner{
		name:             name,
		logger:           logger,
		logObjectStorage: logObjectStorage,
		config:           config,
		tailToken:        make(chan struct{}, 1),
		ingester:         ingester,
		timeline:         NewTimeline(BalanceTransactionsEndpoint, config, state, WithTimelineExpand("data.source")),
	}
}

type Runner struct {
	name             string
	logObjectStorage bridge.LogObjectStorage
	stopChan         chan chan struct{}
	timeline         *timeline
	config           Config
	tailToken        chan struct{}
	logger           sharedlogging.Logger
	ingester         bridge.Ingester[Config, State, *Connector]
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

	batch := bridge.Batch{}
	for _, bt := range ret {
		batchElement, handled := CreateBatchElement(bt, r.name, !tail)
		if !handled {
			r.logger.Errorf("Balance transaction type not handled: %s", bt.Type)
			continue
		}
		if batchElement.Adjustment == nil && batchElement.Payment == nil {
			continue
		}
		batch = append(batch, batchElement)
	}

	err = r.ingester.Ingest(ctx, batch, futureState)
	if err != nil {
		return false, err
	}

	docs := make([]any, 0)
	for _, elem := range ret {
		docs = append(docs, elem)
	}
	if len(docs) > 0 {
		err = r.logObjectStorage.Store(ctx, docs...)
		if err != nil {
			sharedlogging.GetLogger(ctx).Errorf("Unable to record stripe balance transactions: %s", err)
		}
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
					return nil
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
