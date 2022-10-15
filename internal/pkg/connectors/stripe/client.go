package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/numary/payments/internal/pkg/writeonly"

	"github.com/pkg/errors"
	"github.com/stripe/stripe-go/v72"
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
	stripeAccount string
	storage       writeonly.Storage
}

func (d *defaultClient) ForAccount(account string) Client {
	cp := *d
	cp.stripeAccount = account
	return &cp
}

func (d *defaultClient) BalanceTransactions(ctx context.Context, options ...ClientOption) ([]*stripe.BalanceTransaction, bool, error) {
	req, err := http.NewRequest(http.MethodGet, balanceTransactionsEndpoint, nil)
	if err != nil {
		return nil, false, errors.Wrap(err, "creating httphelpers request")
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
	httpResponse, err = d.httpClient.Do(req)
	if err != nil {
		return nil, false, errors.Wrap(err, "doing request")
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("unexpected status code: %d", httpResponse.StatusCode)
	}

	type listResponse struct {
		ListResponse
		Data []json.RawMessage `json:"data"`
	}

	rsp := &listResponse{}
	err = json.NewDecoder(httpResponse.Body).Decode(rsp)
	if err != nil {
		return nil, false, errors.Wrap(err, "decoding response")
	}

	asBalanceTransactions := make([]*stripe.BalanceTransaction, 0)
	if len(rsp.Data) > 0 {
		asMaps := make([]any, 0)

		for _, data := range rsp.Data {
			asMap := make(map[string]interface{})
			err := json.Unmarshal(data, &asMap)
			if err != nil {
				return nil, false, err
			}
			asMaps = append(asMaps, asMap)

			asBalanceTransaction := &stripe.BalanceTransaction{}
			err = json.Unmarshal(data, &asBalanceTransaction)
			if err != nil {
				return nil, false, err
			}
			asBalanceTransactions = append(asBalanceTransactions, asBalanceTransaction)
		}

		err = d.storage.Write(ctx, asMaps...)
		if err != nil {
			return nil, false, err
		}
	}

	return asBalanceTransactions, rsp.HasMore, nil
}

func NewDefaultClient(httpClient *http.Client, apiKey string, storage writeonly.Storage) *defaultClient {
	return &defaultClient{
		httpClient: httpClient,
		apiKey:     apiKey,
		storage:    storage,
	}
}

var _ Client = &defaultClient{}
