package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

const (
	actorClientID = "df05e4b379934cd09963197cc855bfe9"
)

type CreateTemporaryCodeRequest struct {
	UserID       string
	Username     string
	WantedScopes []Scopes
}

type CreateTemporaryCodeResponse struct {
	Code string `json:"code"`
}

func (c *client) CreateTemporaryAuthorizationCode(ctx context.Context, request CreateTemporaryCodeRequest) (CreateTemporaryCodeResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_temporary_code")

	endpoint := fmt.Sprintf("%s/api/v1/oauth/authorization-grant/delegate", c.endpoint)

	scopes := make([]string, len(request.WantedScopes))
	for i, scope := range request.WantedScopes {
		scopes[i] = string(scope)
	}

	form := url.Values{}
	form.Add("external_user_id", request.UserID)
	form.Add("id_hint", request.Username)
	form.Add("actor_client_id", actorClientID) // Constant for tink
	form.Add("scope", strings.Join(scopes, ","))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return CreateTemporaryCodeResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var resp CreateTemporaryCodeResponse
	_, err = c.httpClient.Do(ctx, req, &resp, nil)
	if err != nil {
		return CreateTemporaryCodeResponse{}, err
	}

	return resp, nil
}
