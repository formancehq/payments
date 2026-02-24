package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/formancehq/payments/pkg/connector/metrics"
	"github.com/formancehq/payments/pkg/connector"
)

type recipientsResponse struct {
	Recipients []*Recipient `json:"data"`
}

type Recipient struct {
	ID         string              `json:"id"`
	Attributes RecipientAttributes `json:"attributes"`
}

type RecipientAttributes struct {
	BankAccountCurrency string `json:"bankAccountCurrency"`
	CreatedAt           string `json:"createdAt"`
	BankAccountName     string `json:"bankAccountName"`
}

func (c *client) GetRecipients(ctx context.Context, accountID string, page int, pageSize int) ([]*Recipient, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_recipients")

	endpoint := fmt.Sprintf("%s/accounts/%s/recipients", c.endpoint, accountID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create recipients request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	q := req.URL.Query()
	q.Add("page[size]", strconv.Itoa(pageSize))
	q.Add("page[number]", fmt.Sprint(page))
	q.Add("sortBy", "createdAt.asc")
	req.URL.RawQuery = q.Encode()

	recipients := recipientsResponse{Recipients: make([]*Recipient, 0)}
	var errRes moneycorpErrors
	_, err = c.httpClient.Do(ctx, req, &recipients, &errRes)
	if err != nil {
		return nil, connector.NewWrappedError(
			fmt.Errorf("failed to get recipients: %v", errRes.Error()),
			err,
		)
	}
	return recipients.Recipients, nil
}
