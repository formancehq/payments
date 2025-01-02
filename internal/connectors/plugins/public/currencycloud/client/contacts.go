package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
)

type Contact struct {
	ID string `json:"id"`
}

func (c *client) GetContactID(ctx context.Context, accountID string) (*Contact, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_contacts")

	if err := c.ensureLogin(ctx); err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("account_id", accountID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.buildEndpoint("v2/contacts/find"), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	type Contacts struct {
		Contacts []*Contact `json:"contacts"`
	}

	res := Contacts{Contacts: make([]*Contact, 0)}
	var errRes currencyCloudError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts %w, %w", err, errRes.Error())
	}

	if len(res.Contacts) == 0 {
		return nil, fmt.Errorf("no contact found for account %s: %w", accountID, models.ErrInvalidRequest)
	}

	return res.Contacts[0], nil
}
