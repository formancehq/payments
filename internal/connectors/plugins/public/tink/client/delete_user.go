package client

import (
	"context"
	"fmt"
	"net/http"
)

type DeleteUserRequest struct {
	UserID string
}

func (c *client) DeleteUser(ctx context.Context, req DeleteUserRequest) error {
	authCode, err := c.GetUserAccessToken(ctx, GetUserAccessTokenRequest{
		UserID: req.UserID,
		WantedScopes: []Scopes{
			SCOPES_USER_DELETE,
		},
	})
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/api/v1/user/delete", c.endpoint)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authCode))

	_, err = c.httpClient.Do(ctx, request, nil, nil)
	if err != nil {
		return err
	}

	return nil
}
