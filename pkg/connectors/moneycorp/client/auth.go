package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/payments/pkg/connector/httpwrapper"
	"github.com/formancehq/payments/pkg/connector/metrics"
	"github.com/formancehq/payments/pkg/connector"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Cannot use "golang.org/x/oauth2/clientcredentials" lib because moneycorp
// is only accepting request with "application/json" content type, and the lib
// sets it as application/x-www-form-urlencoded, giving us a 415 error.
type apiTransport struct {
	connectorName string

	clientID string
	apiKey   string
	endpoint string

	accessToken          string
	accessTokenExpiresAt time.Time

	underlying *otelhttp.Transport
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.ensureAccessTokenIsValid(req.Context()); err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+t.accessToken)

	return t.underlying.RoundTrip(req)
}

func (t *apiTransport) ensureAccessTokenIsValid(ctx context.Context) error {
	if t.accessTokenExpiresAt.After(time.Now().Add(5 * time.Second)) {
		return nil
	}

	return t.login(ctx)
}

type loginRequest struct {
	ClientID string `json:"loginId"`
	APIKey   string `json:"apiKey"`
}

type loginResponse struct {
	Data struct {
		AccessToken string `json:"accessToken"`
		ExpiresIn   int    `json:"expiresIn"`
	} `json:"data"`
}

func (t *apiTransport) login(ctx context.Context) error {
	lreq := loginRequest{
		ClientID: t.clientID,
		APIKey:   t.apiKey,
	}

	requestBody, err := json.Marshal(lreq)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	config := &httpwrapper.Config{
		Transport: metrics.NewTransport(t.connectorName, metrics.TransportOpts{}),
	}
	httpClient := httpwrapper.NewClient(config)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		t.endpoint+"/login", bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "authenticate")

	var res loginResponse
	var errRes moneycorpErrors
	statusCode, err := httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return connector.NewWrappedError(
			fmt.Errorf("failed to login: %v", errRes.Error()),
			err,
		)
	}

	if statusCode != http.StatusOK {
		if statusCode >= http.StatusInternalServerError {
			return toError(statusCode, errRes).Error()
		}
		return connector.NewWrappedError(
			fmt.Errorf("failed to login: %v", errRes.Error()),
			toError(statusCode, errRes).Error(),
		)
	}

	t.accessToken = res.Data.AccessToken
	t.accessTokenExpiresAt = time.Now().Add(time.Duration(res.Data.ExpiresIn) * time.Second)
	return nil
}
