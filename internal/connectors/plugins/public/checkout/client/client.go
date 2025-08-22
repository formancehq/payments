package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/checkout/checkout-sdk-go"
	"github.com/checkout/checkout-sdk-go/configuration"
	"github.com/checkout/checkout-sdk-go/nas"
	
	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Client interface {
	GetAccounts(ctx context.Context, page int, pageSize int) ([]*Account, error)
	GetAccountBalances(ctx context.Context) ([]*Balance, error)
	GetExternalAccounts(ctx context.Context, page int, pageSize int) ([]*ExternalAccount, error)
	GetTransactions(ctx context.Context, page, pageSize int) ([]*Transaction, error)
	InitiateTransfer(ctx context.Context, tr *TransferRequest) (*TransferResponse, error)
	InitiatePayout(ctx context.Context, pr *PayoutRequest) (*PayoutResponse, error)
}

type client struct {
	sdk              	*nas.Api
	httpClient 		 	*http.Client
	apiBase			 	string
	apiAuthUrl		 	string
	oauthClientID		string
	oauthClientSecret	string
	entityID 		 	string
}

type acceptHeaderTransport struct {
    base http.RoundTripper
}

func (t *acceptHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    r := req.Clone(req.Context())
    r.Header.Set("Accept", "application/json; schema_version=3.0")
    if r.Header.Get("Content-Type") == "" {
        r.Header.Set("Content-Type", "application/json")
    }
    return t.base.RoundTrip(r)
}

func New(
	env string,
	oauthClientID string,
	oauthClientSecret string,
	entityID string,
) *client {
	var environment configuration.Environment
	switch strings.ToLower(strings.TrimSpace(env)) {
		case "sandbox":
			environment = configuration.Sandbox()
		default:
			environment = configuration.Production()
	} 

	apiBase := environment.BaseUri()
	apiAuthUrl := environment.AuthorizationUri()

	httpClient := &http.Client{
		Transport: &acceptHeaderTransport{
			base: metrics.NewTransport("checkout", metrics.TransportOpts{}),
		},
		Timeout: 30 * time.Second,
	}

	sdk, err := checkout.Builder().
		OAuth().
		WithClientCredentials(strings.TrimSpace(oauthClientID), strings.TrimSpace(oauthClientSecret)).
		WithEnvironment(environment).
		WithHttpClient(httpClient).
		WithScopes(getOAuthScopes()).
		Build()
	if err != nil {
		panic(err)
	}

	return &client{
		sdk:      			sdk,
		httpClient: 		httpClient,
		apiBase: 			apiBase,
		apiAuthUrl:			apiAuthUrl,
		oauthClientID: 		oauthClientID,
		oauthClientSecret: 	oauthClientSecret,
		entityID: 			entityID,
	}
}

func getOAuthScopes() []string {
	return []string{"accounts", "balances", "payments:search"}
}

func (c *client) getAccessToken(ctx context.Context) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("scope", strings.Join(getOAuthScopes(), " "))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(c.apiAuthUrl), strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(strings.TrimSpace(c.oauthClientID), strings.TrimSpace(c.oauthClientSecret))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var apiErr map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return "", fmt.Errorf("oauth token %d: %v", resp.StatusCode, apiErr)
	}

	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", err
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("oauth token missing access_token")
	}
	return tok.AccessToken, nil
}
