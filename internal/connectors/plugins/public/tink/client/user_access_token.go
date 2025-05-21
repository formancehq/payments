package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type GetUserAccessTokenRequest struct {
	UserID       string
	WantedScopes []Scopes
}

type GetUserAccessTokenResponse struct {
	Code string `json:"code"`
}

func (c *client) GetUserAccessToken(ctx context.Context, req GetUserAccessTokenRequest) (string, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_user_access_token")

	endpoint := fmt.Sprintf("%s/api/v1/oauth/authorization-grant", c.endpoint)

	scopes := make([]string, len(req.WantedScopes))
	for i, scope := range req.WantedScopes {
		scopes[i] = string(scope)
	}

	form := url.Values{}
	form.Add("external_user_id", req.UserID)
	form.Add("scope", strings.Join(scopes, ","))

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

	return resp.Code, nil
}
