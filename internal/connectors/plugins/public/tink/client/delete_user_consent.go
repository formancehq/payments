package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type DeleteUserConnectionRequest struct {
	UserID        string
	Username      string
	CredentialsID string
}

func (c *client) DeleteUserConnection(ctx context.Context, req DeleteUserConnectionRequest) error {
	authToken, err := c.getUserAccessToken(ctx, GetUserAccessTokenRequest{
		UserID: req.UserID,
		WantedScopes: []Scopes{
			SCOPES_CREDENTIALS_WRITE,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get user access token: %w", err)
	}

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "delete_user_consent")

	endpoint := fmt.Sprintf("%s/api/v1/credentials/%s", c.endpoint, url.PathEscape(req.CredentialsID))

	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	_, err = c.userClient.Do(ctx, request, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete user consent: %w", err)
	}

	return nil
}
