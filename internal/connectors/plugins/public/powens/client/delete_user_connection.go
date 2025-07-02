package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type DeleteUserConnectionRequest struct {
	AccessToken  string
	ConnectionID string
}

func (c *client) DeleteUserConnection(ctx context.Context, req DeleteUserConnectionRequest) error {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "delete_user_connection")

	endpoint := fmt.Sprintf("%s/2.0/users/me/connections/%s", c.endpoint, req.ConnectionID)
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", req.AccessToken))

	_, err = c.httpClient.Do(ctx, request, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete user connection: %w", err)
	}

	return nil
}
