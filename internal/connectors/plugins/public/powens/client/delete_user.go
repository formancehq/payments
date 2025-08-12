package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type DeleteUserRequest struct {
	AccessToken string
}

func (c *client) DeleteUser(ctx context.Context, req DeleteUserRequest) error {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "delete_user")

	endpoint := fmt.Sprintf("%s/2.0/users/me", c.endpoint)
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", req.AccessToken))

	_, err = c.httpClient.Do(ctx, request, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}
