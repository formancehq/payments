package plaid

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/pkg/connector"
)

func validateDeleteUserRequest(req connector.DeleteUserRequest) error {
	if req.OpenBankingForwardedUser == nil {
		return fmt.Errorf("open banking connections are required: %w", connector.ErrInvalidRequest)
	}

	if req.OpenBankingForwardedUser.Metadata == nil {
		return fmt.Errorf("open banking connections metadata are required: %w", connector.ErrInvalidRequest)
	}

	if _, ok := req.OpenBankingForwardedUser.Metadata[UserTokenMetadataKey]; !ok {
		return fmt.Errorf("missing user token: %w", connector.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) deleteUser(ctx context.Context, req connector.DeleteUserRequest) (connector.DeleteUserResponse, error) {
	if err := validateDeleteUserRequest(req); err != nil {
		return connector.DeleteUserResponse{}, err
	}

	err := p.client.DeleteUser(ctx, req.OpenBankingForwardedUser.Metadata[UserTokenMetadataKey])
	if err != nil {
		return connector.DeleteUserResponse{}, fmt.Errorf("failed to delete user: %w", err)
	}

	return connector.DeleteUserResponse{}, nil
}
