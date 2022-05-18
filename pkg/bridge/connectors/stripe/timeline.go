package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
	"net/url"
	"reflect"
	"time"
)

const (
	apiEndpoint = "https://api.stripe.com/v1"

	EventsEndpoint              = "/events"
	BalanceTransactionsEndpoint = "/balance_transactions"
)

type listResponse struct {
	HasMore bool            `json:"has_more"`
	Data    json.RawMessage `json:"data"`
}

type TimelineOption interface {
	apply(c *timeline)
}
type TimelineOptionFn func(c *timeline)

func (fn TimelineOptionFn) apply(c *timeline) {
	fn(c)
}

func WithTimelineExpand(v ...string) TimelineOptionFn {
	return func(c *timeline) {
		c.expand = v
	}
}

func WithTimelineHttpClient(v *http.Client) TimelineOptionFn {
	return func(c *timeline) {
		c.httpClient = v
	}
}

func WithStartingAt(v time.Time) TimelineOptionFn {
	return func(c *timeline) {
		c.startingAt = v
	}
}

var defaultOptions = []TimelineOption{
	WithTimelineHttpClient(http.DefaultClient),
}

func NewTimeline(endpoint string, cfg Config, state State, options ...TimelineOption) *timeline {
	c := &timeline{
		config:   cfg,
		endpoint: endpoint,
		state:    state,
	}
	options = append(defaultOptions, append([]TimelineOption{
		WithStartingAt(time.Now()),
	}, options...)...)
	for _, opt := range options {
		opt.apply(c)
	}
	return c
}

type timeline struct {
	state                  State
	firstIDAfterStartingAt string
	endpoint               string
	httpClient             *http.Client
	expand                 []string
	startingAt             time.Time
	config                 Config
}

func (tl *timeline) doRequest(ctx context.Context, queryParams url.Values, to interface{}) (bool, error) {

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", apiEndpoint, tl.endpoint), nil)
	if err != nil {
		return false, errors.Wrap(err, "creating http request")
	}

	req = req.WithContext(ctx)
	queryParams.Set("limit", fmt.Sprintf("%d", tl.config.PageSize))
	for _, e := range tl.expand {
		queryParams.Add("expand[]", e)
	}
	req.URL.RawQuery = queryParams.Encode()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(tl.config.ApiKey, "") // gfyrag: really weird authentication right?

	httpResponse, err := tl.httpClient.Do(req)
	if err != nil {
		return false, errors.Wrap(err, "doing request")
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status code: %d", httpResponse.StatusCode)
	}

	rsp := &listResponse{}
	err = json.NewDecoder(httpResponse.Body).Decode(rsp)
	if err != nil {
		return false, errors.Wrap(err, "decoding response")
	}

	err = json.Unmarshal(rsp.Data, to)
	if err != nil {
		return false, errors.Wrap(err, "unmarshalling json response")
	}

	return rsp.HasMore, nil
}

func (tl *timeline) init(ctx context.Context) error {
	type x struct {
		ID string `json:"id"`
	}
	ret := make([]x, 0)
	params := url.Values{}
	params.Set("limit", "1")
	params.Set("created[lt]", fmt.Sprintf("%d", tl.startingAt.Unix()))
	_, err := tl.doRequest(ctx, params, &ret)
	if err != nil {
		return err
	}
	if len(ret) > 0 {
		tl.firstIDAfterStartingAt = reflect.ValueOf(ret).Index(0).FieldByName("ID").String()
	}
	return nil
}

func (tl *timeline) Tail(ctx context.Context, to interface{}) (bool, State, func(), error) {
	queryParams := url.Values{}
	switch {
	case tl.state.OldestID != "":
		queryParams.Set("starting_after", tl.state.OldestID)
	default:
		queryParams.Set("created[lte]", fmt.Sprintf("%d", tl.startingAt.Unix()))
	}

	hasMore, err := tl.doRequest(ctx, queryParams, to)
	if err != nil {
		return false, State{}, nil, err
	}

	futureState := tl.state
	valueOfTo := reflect.ValueOf(to).Elem()
	if valueOfTo.Len() > 0 {
		lastItem := valueOfTo.Index(valueOfTo.Len() - 1)
		futureState.OldestID = lastItem.
			FieldByName("ID").
			String()
	}

	return hasMore, futureState, func() {
		tl.state = futureState
	}, nil
}

func (tl *timeline) Head(ctx context.Context, to interface{}) (bool, State, func(), error) {
	if tl.firstIDAfterStartingAt == "" {
		err := tl.init(ctx)
		if err != nil {
			return false, State{}, nil, err
		}
		if tl.firstIDAfterStartingAt == "" {
			return false, State{}, nil, nil
		}
	}

	queryParams := url.Values{}
	switch {
	case tl.state.MoreRecentID != "":
		queryParams.Set("ending_before", tl.state.MoreRecentID)
	case tl.firstIDAfterStartingAt != "":
		queryParams.Set("ending_before", tl.firstIDAfterStartingAt)
	default:
	}

	hasMore, err := tl.doRequest(ctx, queryParams, to)
	if err != nil {
		return false, State{}, nil, err
	}

	valueOfTo := reflect.ValueOf(to).Elem()
	swap := reflect.Swapper(valueOfTo.Interface())
	for i, j := 0, valueOfTo.Len()-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}

	futureState := tl.state
	if valueOfTo.Len() > 0 {
		futureState.MoreRecentID = valueOfTo.
			Index(0).
			FieldByName("ID").
			String()
	}

	return hasMore, futureState, func() {
		tl.state = futureState
	}, nil
}

func (tl *timeline) State() State {
	return tl.state
}
