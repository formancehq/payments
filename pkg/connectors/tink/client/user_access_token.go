package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/formancehq/payments/pkg/connector/metrics"
)

type GetUserAccessTokenRequest struct {
	UserID       string
	WantedScopes []Scopes
}

type GetUserAccessTokenResponse struct {
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
}

type authorizationGrantResponse struct {
	Code string `json:"code"`
}

func (c *client) getAuthorizationGrantCode(ctx context.Context, userID string, userScopes []Scopes) (string, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_authorization_grant_code")

	endpoint := fmt.Sprintf("%s/api/v1/oauth/authorization-grant", c.endpoint)

	scopes := make([]string, len(userScopes))
	for i, scope := range userScopes {
		scopes[i] = string(scope)
	}

	form := url.Values{}
	form.Add("external_user_id", userID)
	form.Add("scope", strings.Join(scopes, ","))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var resp authorizationGrantResponse
	_, err = c.httpClient.Do(ctx, req, &resp, nil)
	if err != nil {
		return "", err
	}

	return resp.Code, nil
}

func (c *client) getUserAccessToken(ctx context.Context, req GetUserAccessTokenRequest) (string, error) {
	code, err := c.getAuthorizationGrantCode(ctx, req.UserID, req.WantedScopes)
	if err != nil {
		return "", err
	}

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_user_access_token")

	endpoint := fmt.Sprintf("%s/api/v1/oauth/token", c.endpoint)

	form := url.Values{}
	form.Add("client_id", c.clientID)
	form.Add("client_secret", c.clientSecret)
	form.Add("grant_type", "authorization_code")
	form.Add("code", code)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var resp GetUserAccessTokenResponse
	_, err = c.httpClient.Do(ctx, request, &resp, nil)
	if err != nil {
		return "", err
	}

	return resp.AccessToken, nil
}
