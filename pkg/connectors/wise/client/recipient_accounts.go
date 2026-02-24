package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/pkg/connector"
	"github.com/formancehq/payments/pkg/connector/metrics"
)

type RecipientAccountsResponse struct {
	Content                []*RecipientAccount `json:"content"`
	SeekPositionForCurrent uint64              `json:"seekPositionForCurrent"`
	SeekPositionForNext    uint64              `json:"seekPositionForNext"`
	Size                   int                 `json:"size"`
}

type Name struct {
	FullName string `json:"fullName"`
}

type RecipientAccount struct {
	ID       uint64 `json:"id"`
	Profile  uint64 `json:"profileId"`
	Currency string `json:"currency"`
	Name     Name   `json:"name"`
}

func (c *client) GetRecipientAccounts(ctx context.Context, profileID uint64, pageSize int, seekPositionForNext uint64) (*RecipientAccountsResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_recipient_accounts")

	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet, c.endpoint("v2/accounts"), http.NoBody)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("profile", fmt.Sprintf("%d", profileID))
	q.Add("size", fmt.Sprintf("%d", pageSize))
	q.Add("sort", "id,asc")
	if seekPositionForNext > 0 {
		q.Add("seekPosition", fmt.Sprintf("%d", seekPositionForNext))
	}
	req.URL.RawQuery = q.Encode()

	var accounts RecipientAccountsResponse
	var errRes wiseErrors
	statusCode, err := c.httpClient.Do(ctx, req, &accounts, &errRes)
	if err != nil {
		return nil, connector.NewWrappedError(
			fmt.Errorf("failed to get recipient accounts: %v", errRes.Error(statusCode)),
			err,
		)
	}
	return &accounts, nil
}

func (c *client) GetRecipientAccount(ctx context.Context, accountID uint64) (*RecipientAccount, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_recipient_accounts")

	c.mux.Lock()
	defer c.mux.Unlock()
	if rc, ok := c.recipientAccountsCache.Get(accountID); ok {
		return rc, nil
	}

	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet, c.endpoint(fmt.Sprintf("v1/accounts/%d", accountID)), http.NoBody)
	if err != nil {
		return nil, err
	}

	var res RecipientAccount
	var errRes wiseErrors
	statusCode, err := c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		e := errRes.Error(statusCode)
		if e.Code == "RECIPIENT_MISSING" {
			// This is a valid response, we just don't have the account amongst
			// our recipients.
			return &RecipientAccount{}, nil
		}
		return nil, connector.NewWrappedError(
			fmt.Errorf("failed to get recipient account: %v", e.Error()),
			err,
		)
	}

	c.recipientAccountsCache.Add(accountID, &res)
	return &res, nil
}
