package powens

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/pkg/connectors/powens/client"
	"github.com/formancehq/payments/pkg/connector"
)

func validateDeleteUserRequest(req connector.DeleteUserRequest) error {
	if req.OpenBankingForwardedUser == nil {
		return fmt.Errorf("open banking forwarded user is required: %w", connector.ErrInvalidRequest)
	}

	if req.OpenBankingForwardedUser.AccessToken == nil {
		return fmt.Errorf("auth token is required: %w", connector.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) deleteUser(ctx context.Context, req connector.DeleteUserRequest) (connector.DeleteUserResponse, error) {
	if err := validateDeleteUserRequest(req); err != nil {
		return connector.DeleteUserResponse{}, err
	}

	err := p.client.DeleteUser(ctx, client.DeleteUserRequest{
		AccessToken: req.OpenBankingForwardedUser.AccessToken.Token,
	})
	if err != nil {
		return connector.DeleteUserResponse{}, err
	}

	return connector.DeleteUserResponse{}, nil
}
