package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type CreateUserRequest struct {
	ExternalUserID string `json:"external_user_id"`
	Market         string `json:"market"`
	Locale         string `json:"locale"`
}

type CreateUserResponse struct {
	ExternalUserID string `json:"external_user_id"`
	UserID         string `json:"user_id"`
}

func (c *client) CreateUser(ctx context.Context, userID string, market string, locale string) (CreateUserResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_user")

	body, err := json.Marshal(&CreateUserRequest{
		ExternalUserID: userID,
		Market:         market,
		Locale:         locale,
	})
	if err != nil {
		return CreateUserResponse{}, err
	}

	endpoint := fmt.Sprintf("%s/api/v1/user/create", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return CreateUserResponse{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var resp CreateUserResponse
	_, err = c.httpClient.Do(ctx, req, &resp, nil)
	if err != nil {
		return CreateUserResponse{}, fmt.Errorf("failed to create user: %w", err)
	}

	return resp, nil
}
