package stripe

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge"
	"github.com/stripe/stripe-go/v72"
)

type Ingester interface {
	Ingest(ctx context.Context, batch []stripe.BalanceTransaction, commitState TimelineState, tail bool) error
}
type IngesterFn func(ctx context.Context, batch []stripe.BalanceTransaction, commitState TimelineState, tail bool) error

func (fn IngesterFn) Ingest(ctx context.Context, batch []stripe.BalanceTransaction, commitState TimelineState, tail bool) error {
	return fn(ctx, batch, commitState, tail)
}

type defaultIngester struct {
	name             string
	account          string
	logger           sharedlogging.Logger
	ingester         bridge.Ingester[State]
	logObjectStorage bridge.LogObjectStorage
}

func (i *defaultIngester) Ingest(ctx context.Context, txs []stripe.BalanceTransaction, commitState TimelineState, tail bool) error {
	batch := bridge.Batch{}
	for _, bt := range txs {
		batchElement, handled := CreateBatchElement(bt, i.name, !tail)
		if !handled {
			i.logger.Errorf("Balance transaction type not handled: %s", bt.Type)
			continue
		}
		if batchElement.Adjustment == nil && batchElement.Payment == nil {
			continue
		}
		batch = append(batch, batchElement)
	}
	newState := State{}
	if i.account == "" {
		newState.TimelineState = commitState
	} else {
		newState.Accounts = map[string]TimelineState{
			i.account: commitState,
		}
	}
	err := i.ingester.Ingest(ctx, batch, newState)
	if err != nil {
		return err
	}

	docs := make([]any, 0)
	for _, elem := range txs {
		docs = append(docs, elem)
	}
	if len(docs) > 0 {
		err = i.logObjectStorage.Store(ctx, docs...)
		if err != nil {
			sharedlogging.GetLogger(ctx).Errorf("Unable to record stripe balance transactions: %s", err)
		}
	}

	return nil
}

func NewDefaultIngester(
	name string,
	account string,
	logger sharedlogging.Logger,
	ingester bridge.Ingester[State],
	logObjectStorage bridge.LogObjectStorage,
) *defaultIngester {
	return &defaultIngester{
		name:             name,
		account:          account,
		logger:           logger,
		ingester:         ingester,
		logObjectStorage: logObjectStorage,
	}
}
