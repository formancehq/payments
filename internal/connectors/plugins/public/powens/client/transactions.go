package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type PaginationLinks struct {
	Self struct {
		Href string `json:"href"`
	} `json:"self"`
	Next struct {
		Href string `json:"href"`
	} `json:"next"`
	Prev struct {
		Href string `json:"href"`
	} `json:"prev"`
}

type TransactionResponse struct {
	Transactions  []Transaction   `json:"transactions"`
	FirstDate     time.Time       `json:"first_date"`
	LastDate      time.Time       `json:"last_date"`
	ResultMinDate time.Time       `json:"result_min_date"`
	ResultMaxDate time.Time       `json:"result_max_date"`
	Links         PaginationLinks `json:"_links"`
}

func (t *TransactionResponse) UnmarshalJSON(data []byte) error {
	var err error
	type transactionResponse struct {
		Transactions  []Transaction   `json:"transactions"`
		FirstDate     string          `json:"first_date"`
		LastDate      string          `json:"last_date"`
		ResultMinDate string          `json:"result_min_date"`
		ResultMaxDate string          `json:"result_max_date"`
		Links         PaginationLinks `json:"_links"`
	}

	var tr transactionResponse
	if err := json.Unmarshal(data, &tr); err != nil {
		return err
	}

	t.Transactions = tr.Transactions

	if tr.FirstDate != "" {
		t.FirstDate, err = time.Parse(time.DateOnly, tr.FirstDate)
		if err != nil {
			return err
		}
	}

	if tr.LastDate != "" {
		t.LastDate, err = time.Parse(time.DateOnly, tr.LastDate)
		if err != nil {
			return err
		}
	}

	if tr.ResultMinDate != "" {
		t.ResultMinDate, err = time.Parse(time.DateOnly, tr.ResultMinDate)
		if err != nil {
			return err
		}
	}

	if tr.ResultMaxDate != "" {
		t.ResultMaxDate, err = time.Parse(time.DateOnly, tr.ResultMaxDate)
		if err != nil {
			return err
		}
	}
	t.Links = tr.Links

	return nil
}

type Transaction struct {
	ID         int       `json:"id"`
	AccountID  int       `json:"id_account"`
	Date       time.Time `json:"date"`
	DateTime   time.Time `json:"date_time"`
	Value      float64   `json:"value"`
	Type       string    `json:"type"`
	LastUpdate time.Time `json:"last_update"`
}

func (t *Transaction) UnmarshalJSON(data []byte) error {
	var err error
	type transaction struct {
		ID         int     `json:"id"`
		AccountID  int     `json:"id_account"`
		Date       string  `json:"date"`
		DateTime   string  `json:"date_time"`
		Value      float64 `json:"value"`
		Type       string  `json:"type"`
		LastUpdate string  `json:"last_update"`
	}

	var tr transaction
	if err := json.Unmarshal(data, &tr); err != nil {
		return err
	}

	t.ID = tr.ID
	t.AccountID = tr.AccountID

	if tr.Date != "" {
		t.Date, err = time.Parse(time.DateOnly, tr.Date)
		if err != nil {
			return err
		}
	}

	if tr.DateTime != "" {
		t.DateTime, err = time.Parse(time.DateOnly, tr.DateTime)
		if err != nil {
			return err
		}
	}
	t.Value = tr.Value
	t.Type = tr.Type

	if tr.LastUpdate != "" {
		t.LastUpdate, err = time.Parse(time.DateTime, tr.LastUpdate)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *client) ListTransactions(ctx context.Context, accessToken string, lastUpdate time.Time, pageSize int) (TransactionResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_transactions")

	endpoint := fmt.Sprintf("%s/2.0/users/me/transactions", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return TransactionResponse{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	query := req.URL.Query()
	query.Add("last_update", lastUpdate.Format(time.RFC3339))
	query.Add("limit", strconv.Itoa(pageSize))
	req.URL.RawQuery = query.Encode()

	var resp TransactionResponse
	var errResp powensError
	if _, err := c.httpClient.Do(ctx, req, &resp, &errResp); err != nil {
		return TransactionResponse{}, fmt.Errorf("failed to list transactions: %w", errResp.Error())
	}

	return resp, nil
}
