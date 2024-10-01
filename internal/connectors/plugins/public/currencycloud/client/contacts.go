package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Contact struct {
	ID string `json:"id"`
}

func (c *client) GetContactID(ctx context.Context, accountID string) (*Contact, error) {
	// TODO(polo): add metrics
	// f := connectors.ClientMetrics(ctx, "currencycloud", "list_contacts")
	// now := time.Now()
	// defer f(ctx, now)

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
	_, err = c.httpClient.Do(req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts %w, %w", err, errRes.Error())
	}

	if len(res.Contacts) == 0 {
		return nil, fmt.Errorf("no contact found for account %s", accountID)
	}
	return res.Contacts[0], nil
}
