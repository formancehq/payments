package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v2/api"
)

type Account struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Type      string    `json:"type"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}

func (c *client) GetAccounts(ctx context.Context, lastID string, pageSize int64) ([]*Account, string, bool, error) {
	ctx = context.WithValue(ctx, api.MetricOperationContextKey, "list_accounts")

	endpoint := fmt.Sprintf("/accounts?limit=%d", pageSize)
	if lastID != "" {
		endpoint += "&cursor=" + lastID
	}

	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", false, err
	}

	var response struct {
		Data     []*Account `json:"data"`
		NextPage string     `json:"next_page"`
	}
	if err := c.do(req, &response); err != nil {
		return nil, "", false, err
	}

	return response.Data, response.NextPage, response.NextPage != "", nil
}
