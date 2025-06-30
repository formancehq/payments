package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

type CreateUserRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type CreateUserResponse struct {
	AuthToken string `json:"auth_token"`
	Type      string `json:"type"`
	IdUser    int    `json:"id_user"`
	ExpiresIn int64  `json:"expires_in"`
}

func (c *client) CreateUser(ctx context.Context) (CreateUserResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_user")

	body, err := json.Marshal(&CreateUserRequest{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
	})
	if err != nil {
		return CreateUserResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/2.0/auth/init", c.endpoint)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return CreateUserResponse{}, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	var resp CreateUserResponse
	var errResp powensError
	if _, err := c.httpClient.Do(ctx, httpReq, &resp, &errResp); err != nil {
		return CreateUserResponse{}, errorsutils.NewWrappedError(
			fmt.Errorf("failed to create user: %v", errResp.Error()),
			err,
		)
	}

	return resp, nil
}
