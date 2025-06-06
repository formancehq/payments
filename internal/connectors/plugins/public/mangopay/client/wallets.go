package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/formancehq/payments/internal/connectors/metrics"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

type Wallet struct {
	ID           string   `json:"Id"`
	Owners       []string `json:"Owners"`
	Description  string   `json:"Description"`
	CreationDate int64    `json:"CreationDate"`
	Currency     string   `json:"Currency"`
	Balance      struct {
		Currency string      `json:"Currency"`
		Amount   json.Number `json:"Amount"`
	} `json:"Balance"`
}

func (c *client) GetWallets(ctx context.Context, userID string, page, pageSize int) ([]Wallet, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_wallets")

	endpoint := fmt.Sprintf("%s/v2.01/%s/users/%s/wallets", c.endpoint, c.clientID, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}

	q := req.URL.Query()
	q.Add("per_page", strconv.Itoa(pageSize))
	q.Add("page", fmt.Sprint(page))
	q.Add("Sort", "CreationDate:ASC")
	req.URL.RawQuery = q.Encode()

	var wallets []Wallet
	statusCode, err := c.httpClient.Do(ctx, req, &wallets, nil)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get wallets: status code %d", statusCode),
			err,
		)
	}
	return wallets, nil
}

func (c *client) GetWallet(ctx context.Context, walletID string) (*Wallet, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_wallet")

	endpoint := fmt.Sprintf("%s/v2.01/%s/wallets/%s", c.endpoint, c.clientID, walletID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet request: %w", err)
	}

	var wallet Wallet
	statusCode, err := c.httpClient.Do(ctx, req, &wallet, nil)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get wallet: status code %d", statusCode),
			err,
		)
	}
	return &wallet, nil
}
