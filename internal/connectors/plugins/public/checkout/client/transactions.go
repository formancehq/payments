package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type TransactionAction struct {
	Type      string   	 `json:"type"`
}

type Transaction struct {
	ID        				string     			`json:"id"`
	PaymentID 				string     			`json:"payment_id"`
	Type      				string     			`json:"type"`
	Status    				string     			`json:"status"`
	Amount    				int64      			`json:"amount"`
	Currency  				string     			`json:"currency"`
	Scheme    				string     			`json:"scheme"`
	SourceAccountReference 	string 				`json:"sourceAccountReference"`
	Actions	  			   	[]TransactionAction `json:"actions"`
	CreatedAt 				time.Time  			`json:"created_at"`
}

type searchSort struct {
	Field string `json:"field"`
	Order string `json:"order"`
}
type searchFilters struct {
	EntityIDs []string `json:"entity_ids,omitempty"`
}
type searchPaymentsRequest struct {
	Query   string         `json:"query,omitempty"`
	From    int            `json:"from,omitempty"`
	Limit   int            `json:"limit,omitempty"`
	Sort    []searchSort   `json:"sort,omitempty"`
	Filters *searchFilters `json:"filters,omitempty"`
}
type searchPaymentsResponse struct {
	Data []struct {
		ID          string    			`json:"id"`
		Amount      int64     			`json:"amount"`
		Currency    string    			`json:"currency"`
		Status      string    			`json:"status"`
		Approved    bool      			`json:"approved"`
		Source      struct {
			Scheme      string    `json:"scheme"`
		} `json:"source"`
		RequestedOn time.Time 			`json:"requested_on"`
		Actions		[]TransactionAction `json:"actions"`
	} `json:"data"`
}

func (c *client) GetTransactions(ctx context.Context, page, pageSize int) ([]*Transaction, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_transactions")

	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 100
	}

	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("oauth token: %w", err)
	}

	reqBody := searchPaymentsRequest{
		Query: "",
		Limit: pageSize,
	}
	body, _ := json.Marshal(reqBody)

	url := strings.TrimRight(c.apiBase, "/") + "/payments/search"

	fmt.Sprintf("PAYMENTS URL : %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create search request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json; schema_version=3.0")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search payments http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var apiErr map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return nil, fmt.Errorf("search payments %d: %v", resp.StatusCode, apiErr)
	}

	var sr searchPaymentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("decode search payments: %w", err)
	}

	transactions := make([]*Transaction, 0, len(sr.Data))
	for _, it := range sr.Data {
		transactions = append(transactions, &Transaction{
			ID:        it.ID,
			PaymentID: it.ID,
			Type:      "payment",
			Scheme:    it.Source.Scheme,
			Status:    it.Status,
			Amount:    it.Amount,
			Currency:  it.Currency,
			SourceAccountReference: c.entityID,
			Actions:   it.Actions,
			CreatedAt: it.RequestedOn,
		})
	}

	return transactions, nil
}
