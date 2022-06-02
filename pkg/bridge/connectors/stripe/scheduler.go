package stripe

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge"
	"github.com/stripe/stripe-go/v72"
	"sync"
)

type Scheduler struct {
	logObjectStorage bridge.LogObjectStorage
	runner           *Runner
	accountTriggers  map[string]*timelineTrigger
	logger           sharedlogging.Logger
	triggersLock     sync.RWMutex
	stateLock        sync.Mutex
	state            State
	ingester         bridge.Ingester[State]
	config           Config
	timelineOptions  []TimelineOption
	client           Client
}

func (s *Scheduler) Name() string {
	return "stripe"
}

func (s *Scheduler) accountLogger(account string) sharedlogging.Logger {
	if account == "" {
		return s.logger
	}
	return s.logger.WithFields(map[string]interface{}{
		"account": account,
	})
}

func (s *Scheduler) createTrigger(account string, state TimelineState) *timelineTrigger {
	s.triggersLock.Lock()
	defer s.triggersLock.Unlock()

	s.accountLogger(account).Infof("Create new trigger")
	trigger := NewTimelineTrigger(
		s.logger.WithFields(map[string]interface{}{
			"component": "trigger",
			"timeline":  account,
		}),
		s.ingesterFor(account),
		NewTimeline(s.client, s.config.TimelineConfig, state, s.timelineOptions...),
	)
	if account != "" {
		if s.accountTriggers == nil {
			s.accountTriggers = make(map[string]*timelineTrigger)
		}
		s.accountTriggers[account] = trigger
	}
	return trigger
}

func (s *Scheduler) triggerFetch(account string) {
	s.triggersLock.RLock()
	trigger, ok := s.accountTriggers[account]
	s.triggersLock.RUnlock()

	if !ok {
		trigger = s.createTrigger(account, s.state.Accounts[account])
	}

	go func() {
		err := trigger.Fetch(context.Background())
		if err != nil {
			s.logger.Errorf("Error triggering connected account fetching: %s", err)
		}
	}()
}

func (s *Scheduler) ingest(ctx context.Context, bts []*stripe.BalanceTransaction, account string, commitState TimelineState, tail bool) error {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	connectedAccounts := make([]string, 0)

	batch := bridge.Batch{}
	for _, bt := range bts {
		batchElement, handled := CreateBatchElement(bt, !tail)
		if !handled {
			s.accountLogger(account).Errorf("Balance transaction type not handled: %s", bt.Type)
			continue
		}
		if batchElement.Adjustment == nil && batchElement.Payment == nil {
			continue
		}
		batch = append(batch, batchElement)
		if bt.Type == "transfer" {
			connectedAccounts = append(connectedAccounts, bt.Source.Transfer.Destination.ID)
		}
	}
	s.accountLogger(account).WithFields(map[string]interface{}{
		"state": commitState,
	}).Debugf("updating state")
	newState := s.state
	if account == "" {
		newState.TimelineState = commitState
	} else {
		if newState.Accounts == nil {
			newState.Accounts = map[string]TimelineState{}
		}
		newState.Accounts[account] = commitState
	}

	for _, ca := range connectedAccounts {
		_, ok := newState.Accounts[ca]
		if !ok {
			if newState.Accounts == nil {
				newState.Accounts = map[string]TimelineState{}
			}
			newState.Accounts[ca] = TimelineState{}
			s.triggerFetch(ca)
		}
	}

	err := s.ingester.Ingest(ctx, batch, newState)
	if err != nil {
		return err
	}

	s.state = newState

	docs := make([]any, 0)
	for _, elem := range bts {
		docs = append(docs, elem)
	}
	if len(docs) > 0 {
		err = s.logObjectStorage.Store(ctx, docs...)
		if err != nil {
			s.accountLogger(account).Errorf("Unable to record stripe balance transactions: %s", err)
		}
	}

	return nil
}

func (i *Scheduler) ingesterFor(account string) Ingester {
	return IngesterFn(func(ctx context.Context, batch []*stripe.BalanceTransaction, commitState TimelineState, tail bool) error {
		return i.ingest(ctx, batch, account, commitState, tail)
	})
}

func (s *Scheduler) Start(ctx context.Context) error {

	s.runner = NewRunner(
		s.logger.WithFields(map[string]interface{}{
			"component": "runner",
			"timeline":  "main",
		}),
		s.createTrigger("", s.state.TimelineState),
		s.config.PollingPeriod,
	)

	go func() {
		err := s.runner.Run(ctx)
		if err != nil {
			panic(err)
		}
	}()

	if s.state.Accounts != nil {
		for account := range s.state.Accounts {
			s.triggerFetch(account)
		}
	}

	return nil
}

func (s *Scheduler) Stop(ctx context.Context) error {
	s.logger.Infof("Stopping...")
	if s.runner != nil {
		s.logger.Infof("Stopping main runner...")
		err := s.runner.Stop(ctx)
		if err != nil {
			return err
		}
		s.runner = nil
		s.logger.Infof("Main runner stopped!")
	}
	wg := sync.WaitGroup{}
	wg.Add(len(s.accountTriggers))
	for account, trigger := range s.accountTriggers {
		go func(trigger *timelineTrigger) {
			defer wg.Done()
			logger := s.logger.WithFields(map[string]any{
				"account": account,
			})
			logger.Infof("Stopping account trigger...")
			trigger.Cancel(ctx)
			delete(s.accountTriggers, account)
			logger.Infof("Trigger stopped")
		}(trigger)
	}
	wg.Wait()
	return nil
}

func NewScheduler(
	logObjectStorage bridge.LogObjectStorage,
	logger sharedlogging.Logger,
	ingester bridge.Ingester[State],
	client Client,
	cfg Config,
	state State,
	opts ...TimelineOption) *Scheduler {
	return &Scheduler{
		logObjectStorage: logObjectStorage,
		logger: logger.WithFields(map[string]any{
			"component": "scheduler",
		}),
		accountTriggers: map[string]*timelineTrigger{},
		ingester:        ingester,
		config:          cfg,
		state:           state,
		timelineOptions: opts,
		client:          client,
	}
}
