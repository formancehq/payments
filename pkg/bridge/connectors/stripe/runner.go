package stripe

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge"
	"github.com/pkg/errors"
	"github.com/stripe/stripe-go/v72"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type commandHolder struct {
	command func()
	done    chan struct{}
}

func NewRunner(db *mongo.Database, logger sharedlogging.Logger, ingester bridge.Ingester[Config, State, *Connector], config Config, state State) *Runner {
	return &Runner{
		logger:    logger,
		db:        db,
		config:    config,
		commands:  make(chan commandHolder),
		tailToken: make(chan struct{}, 1),
		ingester:  ingester,
		timeline:  NewTimeline(BalanceTransactionsEndpoint, config, state, WithTimelineExpand("data.source")),
	}
}

type page struct {
	tail    bool
	hasMore bool
	err     error
}

type Runner struct {
	db        *mongo.Database
	stopChan  chan chan struct{}
	timeline  *timeline
	commands  chan commandHolder
	config    Config
	tailToken chan struct{}
	logger    sharedlogging.Logger
	ingester  bridge.Ingester[Config, State, *Connector]
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
		batchElement := CreateBatchElement(bt, !tail)
		if batchElement.Adjustment == nil && batchElement.Payment == nil {
			continue
		}
		batch = append(batch, batchElement)
	}

	err = r.ingester.Ingest(ctx, batch, futureState)
	if err != nil {
		return false, err
	}

	// TODO: Recordings all stripe balance transaction for debug purpose
	// This will be removed in a later version
	docs := make([]interface{}, 0)
	for _, elem := range ret {
		docs = append(docs, elem)
	}
	if len(docs) > 0 {
		_, err = r.db.Collection("StripeBalanceTransaction").InsertMany(ctx, docs)
		if err != nil {
			sharedlogging.GetLogger(ctx).Errorf("Unable to record stripe balance transactions: %s", err)
		}
	}

	commitFn()
	return hasMore, nil
}

func (r *Runner) doCmd(ctx context.Context, fn func()) error {
	doneCh := make(chan struct{})
	r.commands <- commandHolder{command: fn, done: doneCh}
	select {
	case <-doneCh:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
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

	for {
		select { // Add a dedicated select to handle commands. It allow command to be executed in priority.
		case cmd := <-r.commands:
			cmd.command()
			close(cmd.done)
		case ch := <-r.stopChan:
			close(ch)
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
				case cmd := <-r.commands:
					cmd.command()
					close(cmd.done)
				default:
					// Nothing to do
				}
			}
			resetTimer()
		default:
			select {
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
