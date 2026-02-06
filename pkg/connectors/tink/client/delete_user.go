package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/pkg/connector/metrics"
)

type DeleteUserRequest struct {
	UserID string
}

func (c *client) DeleteUser(ctx context.Context, req DeleteUserRequest) error {
	authCode, err := c.getUserAccessToken(ctx, GetUserAccessTokenRequest{
		UserID: req.UserID,
		WantedScopes: []Scopes{
			SCOPES_USER_DELETE,
		},
	})
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "delete_user")
	endpoint := fmt.Sprintf("%s/api/v1/user/delete", c.endpoint)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authCode))

	_, err = c.userClient.Do(ctx, request, nil, nil)
	if err != nil {
		return err
	}

	return nil
}
