package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/stripe/stripe-go/v72"
	"net/http"
)

type ClientOption interface {
	apply(req *http.Request)
}
type ClientOptionFn func(req *http.Request)

func (fn ClientOptionFn) apply(req *http.Request) {
	fn(req)
}

func QueryParam(key string, value string) ClientOptionFn {
	return func(req *http.Request) {
		q := req.URL.Query()
		q.Set(key, value)
		req.URL.RawQuery = q.Encode()
	}
}

type Client interface {
	BalanceTransactions(ctx context.Context, options ...ClientOption) ([]*stripe.BalanceTransaction, bool, error)
	ForAccount(account string) Client
}

type defaultClient struct {
	httpClient    *http.Client
	apiKey        string
	pool          *pool
	stripeAccount string
}

func (d *defaultClient) ForAccount(account string) Client {
	cp := *d
	cp.stripeAccount = account
	return &cp
}

func (d *defaultClient) BalanceTransactions(ctx context.Context, options ...ClientOption) ([]*stripe.BalanceTransaction, bool, error) {
	req, err := http.NewRequest(http.MethodGet, balanceTransactionsEndpoint, nil)
	if err != nil {
		return nil, false, errors.Wrap(err, "creating http request")
	}

	req = req.WithContext(ctx)

	for _, opt := range options {
		opt.apply(req)
	}
	if d.stripeAccount != "" {
		req.Header.Set("Stripe-Account", d.stripeAccount)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(d.apiKey, "") // gfyrag: really weird authentication right?

	var httpResponse *http.Response
	err = d.pool.Push(ctx, func(ctx context.Context) error {
		httpResponse, err = d.httpClient.Do(req)
		return err
	})
	if err != nil {
		return nil, false, errors.Wrap(err, "doing request")
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("unexpected status code: %d", httpResponse.StatusCode)
	}

	rsp := &ListResponse{}
	err = json.NewDecoder(httpResponse.Body).Decode(rsp)
	if err != nil {
		return nil, false, errors.Wrap(err, "decoding response")
	}

	return rsp.Data, rsp.HasMore, nil
}

func NewDefaultClient(httpClient *http.Client, pool *pool, apiKey string) *defaultClient {
	return &defaultClient{
		httpClient: httpClient,
		apiKey:     apiKey,
		pool:       pool,
	}
}

var _ Client = &defaultClient{}
