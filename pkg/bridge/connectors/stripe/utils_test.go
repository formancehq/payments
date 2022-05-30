package stripe

import (
	"context"
	"flag"
	"fmt"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/go-libs/sharedlogging/sharedlogginglogrus"
	"github.com/sirupsen/logrus"
	"github.com/stripe/stripe-go/v72"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Verbose() {
		l := logrus.New()
		l.Level = logrus.DebugLevel
		sharedlogging.SetFactory(sharedlogging.StaticLoggerFactory(sharedlogginglogrus.New(l)))
	}

	os.Exit(m.Run())
}

type clientMockExpectation struct {
	query   url.Values
	hasMore bool
	items   []*stripe.BalanceTransaction
}

func (e *clientMockExpectation) QueryParam(key string, value any) *clientMockExpectation {
	var qpvalue string
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		qpvalue = fmt.Sprintf("%d", value)
	default:
		qpvalue = fmt.Sprintf("%s", value)
	}
	e.query.Set(key, qpvalue)
	return e
}

func (e *clientMockExpectation) StartingAfter(v string) *clientMockExpectation {
	e.QueryParam("starting_after", v)
	return e
}

func (e *clientMockExpectation) CreatedLte(v time.Time) *clientMockExpectation {
	e.QueryParam("created[lte]", v.Unix())
	return e
}

func (e *clientMockExpectation) Limit(v int) *clientMockExpectation {
	e.QueryParam("limit", v)
	return e
}

func (e *clientMockExpectation) RespondsWith(hasMore bool, txs ...*stripe.BalanceTransaction) *clientMockExpectation {
	e.hasMore = hasMore
	e.items = txs
	return e
}

func (e *clientMockExpectation) handle(ctx context.Context, options ...ClientOption) ([]*stripe.BalanceTransaction, bool, error) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, o := range options {
		o.apply(req)
	}
	for k := range e.query {
		if req.URL.Query().Get(k) != e.query.Get(k) {
			return nil, false, fmt.Errorf("mismatch query params, expected query param '%s' with value '%s', got '%s'", k, e.query.Get(k), req.URL.Query().Get(k))
		}
	}
	if !reflect.DeepEqual(req.URL.Query(), e.query) {
	}
	return e.items, e.hasMore, nil
}

type clientMock struct {
	expectations *FIFO[*clientMockExpectation]
}

func (m *clientMock) BalanceTransactions(ctx context.Context, options ...ClientOption) ([]*stripe.BalanceTransaction, bool, error) {
	e, ok := m.expectations.Pop()
	if !ok {
		return nil, false, fmt.Errorf("no more expectation")
	}

	return e.handle(ctx, options...)
}

func (m *clientMock) Expect() *clientMockExpectation {
	e := &clientMockExpectation{
		query: url.Values{},
	}
	m.expectations.Push(e)
	return e
}

func NewClientMock(t *testing.T) *clientMock {
	m := &clientMock{
		expectations: &FIFO[*clientMockExpectation]{},
	}
	t.Cleanup(func() {
		if !m.expectations.Empty() && !t.Failed() {
			t.Errorf("all expectations not consumed")
		}
	})
	return m
}

var _ Client = &clientMock{}

type FIFO[ITEM any] struct {
	mu    sync.Mutex
	items []ITEM
}

func (s *FIFO[ITEM]) Pop() (ret ITEM, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.items) == 0 {
		var z ITEM
		return z, false
	}
	ret = s.items[0]
	ok = true
	if len(s.items) == 1 {
		s.items = make([]ITEM, 0)
		return
	}
	s.items = s.items[1:]
	return
}

func (s *FIFO[ITEM]) Push(i ITEM) *FIFO[ITEM] {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = append(s.items, i)
	return s
}

func (s *FIFO[ITEM]) Empty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items) == 0
}
