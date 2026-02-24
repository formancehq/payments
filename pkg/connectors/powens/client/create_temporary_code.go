package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/pkg/connector/metrics"
	"github.com/formancehq/payments/pkg/connector"
)

type CreateTemporaryLinkRequest struct {
	AccessToken string
}

type CreateTemporaryLinkResponse struct {
	Code      string `json:"code"`
	Type      string `json:"type"`
	Access    string `json:"access"`
	ExpiresIn int    `json:"expires_in"`
}

func (c *client) CreateTemporaryCode(ctx context.Context, request CreateTemporaryLinkRequest) (CreateTemporaryLinkResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_temporary_link_code")

	endpoint := fmt.Sprintf("%s/2.0/auth/token/code", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return CreateTemporaryLinkResponse{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", request.AccessToken))

	var resp CreateTemporaryLinkResponse
	var errResp powensError
	if _, err := c.httpClient.Do(ctx, req, &resp, &errResp); err != nil {
		return CreateTemporaryLinkResponse{}, connector.NewWrappedError(
			fmt.Errorf("failed to create temporary link code: %v", errResp.Error()),
			err,
		)
	}

	return resp, nil
}
