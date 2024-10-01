package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
)

func (c *client) Authenticate(ctx context.Context) error {
	// TODO(polo): metrics
	// f := connectors.ClientMetrics(ctx, "currencycloud", "authenticate")
	// now := time.Now()
	// defer f(ctx, now)

	form := make(url.Values)

	form.Add("login_id", c.loginID)
	form.Add("api_key", c.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.buildEndpoint("v2/authenticate/api"), strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")

	//nolint:tagliatelle // allow for client code
	type response struct {
		AuthToken string `json:"auth_token"`
	}

	var res response
	var errRes currencyCloudError
	_, err = c.httpClient.Do(req, &res, &errRes)
	switch err {
	case nil:
		c.authToken = res.AuthToken
	case httpwrapper.ErrStatusCodeUnexpected:
		// TODO(polo): retryable errors
		return errRes.Error()
	}

	return nil
}
