package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
)

type Payout struct {
	ID             uint64      `json:"id"`
	Reference      string      `json:"reference"`
	Status         string      `json:"status"`
	SourceAccount  uint64      `json:"sourceAccount"`
	SourceCurrency string      `json:"sourceCurrency"`
	SourceValue    json.Number `json:"sourceValue"`
	TargetAccount  uint64      `json:"targetAccount"`
	TargetCurrency string      `json:"targetCurrency"`
	TargetValue    json.Number `json:"targetValue"`
	Business       uint64      `json:"business"`
	Created        string      `json:"created"`
	//nolint:tagliatelle // allow for clients
	CustomerTransactionID string `json:"customerTransactionId"`
	Details               struct {
		Reference string `json:"reference"`
	} `json:"details"`
	Rate float64 `json:"rate"`
	User uint64  `json:"user"`

	SourceBalanceID      uint64 `json:"-"`
	DestinationBalanceID uint64 `json:"-"`

	CreatedAt time.Time `json:"-"`
}

func (t *Payout) UnmarshalJSON(data []byte) error {
	type Alias Transfer

	aux := &struct {
		Created string `json:"created"`
		*Alias
	}{
		Alias: (*Alias)(t),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var err error

	t.CreatedAt, err = time.Parse("2006-01-02 15:04:05", aux.Created)
	if err != nil {
		return fmt.Errorf("failed to parse created time: %w", err)
	}

	return nil
}

func (c *client) GetPayout(ctx context.Context, payoutID string) (*Payout, error) {
	ctx = context.WithValue(ctx, httpwrapper.MetricOperationContextKey, "get_payout")

	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet, c.endpoint("v1/transfers/"+payoutID), http.NoBody)
	if err != nil {
		return nil, err
	}

	var payout Payout
	var errRes wiseErrors
	statusCode, err := c.httpClient.Do(ctx, req, &payout, &errRes)
	if err != nil {
		return &payout, fmt.Errorf("failed to get payout: %w %w", err, errRes.Error(statusCode).Error())
	}
	return &payout, nil
}

func (c *client) CreatePayout(ctx context.Context, quote Quote, targetAccount uint64, transactionID string) (*Payout, error) {
	// TODO(polo): metrics
	ctx = context.WithValue(ctx, httpwrapper.MetricOperationContextKey, "initiate_payout")
	// now := time.Now()
	// defer f(ctx, now)

	reqBody, err := json.Marshal(map[string]interface{}{
		"targetAccount":         targetAccount,
		"quoteUuid":             quote.ID.String(),
		"customerTransactionId": transactionID,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("v1/transfers"), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	var payout Payout
	var errRes wiseErrors
	statusCode, err := c.httpClient.Do(ctx, req, &payout, &errRes)
	if err != nil {
		return &payout, fmt.Errorf("failed to make payout: %w %w", err, errRes.Error(statusCode).Error())
	}
	return &payout, nil
}
