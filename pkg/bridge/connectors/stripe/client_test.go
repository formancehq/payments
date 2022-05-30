package stripe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stripe/stripe-go/v72"
	"io/ioutil"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"
)

type httpMockExpectation interface {
	handle(t *testing.T, r *http.Request) (*http.Response, error)
}

type httpMock struct {
	t            *testing.T
	expectations []httpMockExpectation
	mu           sync.Mutex
}

func NewHTTPMock(t *testing.T) (*httpMock, *http.Client) {
	m := &httpMock{
		t:            t,
		expectations: []httpMockExpectation{},
	}
	return m, &http.Client{
		Transport: m,
	}
}

func (m *httpMock) RoundTrip(request *http.Request) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.expectations) == 0 {
		return nil, fmt.Errorf("no more expectations")
	}

	e := m.expectations[0]
	if len(m.expectations) == 1 {
		m.expectations = make([]httpMockExpectation, 0)
	} else {
		m.expectations = m.expectations[1:]
	}

	return e.handle(m.t, request)
}

var _ http.RoundTripper = &httpMock{}

type httpExpect[REQUEST any, RESPONSE any] struct {
	statusCode   int
	path         string
	method       string
	requestBody  *REQUEST
	responseBody *RESPONSE
	queryParams  map[string]any
}

func (e *httpExpect[REQUEST, RESPONSE]) handle(t *testing.T, request *http.Request) (*http.Response, error) {

	if e.path != request.URL.Path {
		return nil, fmt.Errorf("expected url was '%s', got, '%s'", e.path, request.URL.Path)
	}
	if e.method != request.Method {
		return nil, fmt.Errorf("expected method was '%s', got, '%s'", e.method, request.Method)
	}
	if e.requestBody != nil {
		body := new(REQUEST)
		err := json.NewDecoder(request.Body).Decode(body)
		if err != nil {
			panic(err)
		}
		if !reflect.DeepEqual(*e.responseBody, *body) {
			return nil, fmt.Errorf("mismatch body")
		}
	}

	for key, value := range e.queryParams {
		qpvalue := ""
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			qpvalue = fmt.Sprintf("%d", value)
		default:
			qpvalue = fmt.Sprintf("%s", value)
		}
		if rvalue := request.URL.Query().Get(key); rvalue != qpvalue {
			return nil, fmt.Errorf("expected query param '%s' with value '%s', got '%s'", key, qpvalue, rvalue)
		}
	}

	data := make([]byte, 0)
	if e.responseBody != nil {
		var err error
		data, err = json.Marshal(e.responseBody)
		if err != nil {
			panic(err)
		}
	}

	return &http.Response{
		StatusCode:    e.statusCode,
		Body:          ioutil.NopCloser(bytes.NewReader(data)),
		ContentLength: int64(len(data)),
		Request:       request,
	}, nil
}

func (e *httpExpect[REQUEST, RESPONSE]) Path(p string) *httpExpect[REQUEST, RESPONSE] {
	e.path = p
	return e
}

func (e *httpExpect[REQUEST, RESPONSE]) Method(p string) *httpExpect[REQUEST, RESPONSE] {
	e.method = p
	return e
}

func (e *httpExpect[REQUEST, RESPONSE]) Body(body *REQUEST) *httpExpect[REQUEST, RESPONSE] {
	e.requestBody = body
	return e
}

func (e *httpExpect[REQUEST, RESPONSE]) QueryParam(key string, value any) *httpExpect[REQUEST, RESPONSE] {
	e.queryParams[key] = value
	return e
}

func (e *httpExpect[REQUEST, RESPONSE]) RespondsWith(statusCode int, body *RESPONSE) *httpExpect[REQUEST, RESPONSE] {
	e.statusCode = statusCode
	e.responseBody = body
	return e
}

func Expect[REQUEST any, RESPONSE any](mock *httpMock) *httpExpect[REQUEST, RESPONSE] {
	e := &httpExpect[REQUEST, RESPONSE]{
		queryParams: map[string]any{},
	}
	mock.mu.Lock()
	defer mock.mu.Unlock()

	mock.expectations = append(mock.expectations, e)
	return e
}

type stripeBalanceTransactionListExpect struct {
	*httpExpect[struct{}, MockedListResponse]
}

func (e *stripeBalanceTransactionListExpect) Path(p string) *stripeBalanceTransactionListExpect {
	e.httpExpect.Path(p)
	return e
}

func (e *stripeBalanceTransactionListExpect) Method(p string) *stripeBalanceTransactionListExpect {
	e.httpExpect.Method(p)
	return e
}

func (e *stripeBalanceTransactionListExpect) QueryParam(key string, value any) *stripeBalanceTransactionListExpect {
	e.httpExpect.QueryParam(key, value)
	return e
}

func (e *stripeBalanceTransactionListExpect) RespondsWith(statusCode int, hasMore bool, body ...*stripe.BalanceTransaction) *stripeBalanceTransactionListExpect {
	e.httpExpect.RespondsWith(statusCode, &MockedListResponse{
		HasMore: hasMore,
		Data:    body,
	})
	return e
}

func (e *stripeBalanceTransactionListExpect) StartingAfter(v string) *stripeBalanceTransactionListExpect {
	e.QueryParam("starting_after", v)
	return e
}

func (e *stripeBalanceTransactionListExpect) CreatedLte(v time.Time) *stripeBalanceTransactionListExpect {
	e.QueryParam("created[lte]", v.Unix())
	return e
}

func (e *stripeBalanceTransactionListExpect) Limit(v int) *stripeBalanceTransactionListExpect {
	e.QueryParam("limit", v)
	return e
}

func ExpectBalanceTransactionList(mock *httpMock) *stripeBalanceTransactionListExpect {
	e := Expect[struct{}, MockedListResponse](mock)
	e.Path("/v1/balance_transactions").Method(http.MethodGet)
	return &stripeBalanceTransactionListExpect{
		httpExpect: e,
	}
}

func DatePtr(t time.Time) *time.Time {
	return &t
}

type BalanceTransactionSource stripe.BalanceTransactionSource

func (t *BalanceTransactionSource) MarshalJSON() ([]byte, error) {
	type Aux BalanceTransactionSource
	return json.Marshal(struct {
		Aux
		Charge   *stripe.Charge   `json:"charge"`
		Payout   *stripe.Payout   `json:"payout"`
		Refund   *stripe.Refund   `json:"refund"`
		Transfer *stripe.Transfer `json:"transfer"`
	}{
		Aux:      Aux(*t),
		Charge:   t.Charge,
		Payout:   t.Payout,
		Refund:   t.Refund,
		Transfer: t.Transfer,
	})
}

type BalanceTransaction stripe.BalanceTransaction

func (t *BalanceTransaction) MarshalJSON() ([]byte, error) {
	type Aux BalanceTransaction
	return json.Marshal(struct {
		Aux
		Source *BalanceTransactionSource `json:"source"`
	}{
		Aux:    Aux(*t),
		Source: (*BalanceTransactionSource)(t.Source),
	})
}

type MockedListResponse struct {
	HasMore bool                         `json:"has_more"`
	Data    []*stripe.BalanceTransaction `json:"data"`
}

func (t *MockedListResponse) MarshalJSON() ([]byte, error) {
	type Aux MockedListResponse

	txs := make([]*BalanceTransaction, 0)
	for _, tx := range t.Data {
		txs = append(txs, (*BalanceTransaction)(tx))
	}

	return json.Marshal(struct {
		Aux
		Data []*BalanceTransaction `json:"data"`
	}{
		Aux:  Aux(*t),
		Data: txs,
	})
}
