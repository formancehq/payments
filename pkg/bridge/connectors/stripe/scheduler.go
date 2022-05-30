package stripe

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge"
	"github.com/stripe/stripe-go/v72"
	"sync"
)

type Scheduler struct {
	name             string
	logObjectStorage bridge.LogObjectStorage
	runner           *Runner
	accountRunners   map[string]*Runner
	logger           sharedlogging.Logger
	runnersLock      sync.RWMutex
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

func (s *Scheduler) createRunner(account string, cfg Config, state TimelineState) {

	s.runnersLock.Lock()
	defer s.runnersLock.Unlock()

	logger := s.logger.WithFields(map[string]interface{}{
		"account": account,
	})

	logger.Infof("Create new runner")

	runner := NewRunner(
		logger.WithFields(map[string]interface{}{
			"component": "runner",
		}),
		s.ingesterFor(account),
		NewTimeline(s.client, cfg.TimelineConfig, state, append(s.timelineOptions, WithTimelineExpand("data.source"))...),
		cfg.PollingPeriod,
	)
	s.accountRunners[account] = runner

	go func(runner *Runner) {
		err := runner.Run(context.Background())
		if err != nil {
			panic(err)
		}
	}(runner)
}

func (s *Scheduler) ingest(ctx context.Context, bts []*stripe.BalanceTransaction, account string, commitState TimelineState, tail bool) error {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	connectedAccounts := make([]string, 0)

	batch := bridge.Batch{}
	for _, bt := range bts {
		batchElement, handled := CreateBatchElement(bt, s.name, !tail)
		if !handled {
			s.logger.Errorf("Balance transaction type not handled: %s", bt.Type)
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
			newState.Accounts[ca] = TimelineState{}
			s.createRunner(ca, s.config, TimelineState{})
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
			sharedlogging.GetLogger(ctx).Errorf("Unable to record stripe balance transactions: %s", err)
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
		s.ingesterFor(""),
		NewTimeline(s.client, s.config.TimelineConfig, s.state.TimelineState, append(s.timelineOptions, WithTimelineExpand("data.source"))...),
		s.config.PollingPeriod,
	)

	go func() {
		err := s.runner.Run(ctx)
		if err != nil {
			panic(err)
		}
	}()

	if s.state.Accounts != nil {
		for account, accountState := range s.state.Accounts {
			s.createRunner(account, s.config, accountState)
		}
	}

	return nil
}

func (s *Scheduler) Stop(ctx context.Context) error {
	if s.runner != nil {
		err := s.runner.Stop(ctx)
		if err != nil {
			return err
		}
		s.runner = nil
	}
	for account, runner := range s.accountRunners {
		err := runner.Stop(ctx)
		if err != nil {
			return err
		}
		delete(s.accountRunners, account)
	}
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
		logger:           logger,
		accountRunners:   map[string]*Runner{},
		ingester:         ingester,
		config:           cfg,
		state:            state,
		timelineOptions:  opts,
		client:           client,
	}
}
