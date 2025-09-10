package plaid

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

func validateDeleteUserRequest(req models.DeleteUserRequest) error {
	if req.OpenBankingForwardedUser == nil {
		return fmt.Errorf("open banking connections are required: %w", models.ErrInvalidRequest)
	}

	if req.OpenBankingForwardedUser.Metadata == nil {
		return fmt.Errorf("open banking connections metadata are required: %w", models.ErrInvalidRequest)
	}

	if _, ok := req.OpenBankingForwardedUser.Metadata[UserTokenMetadataKey]; !ok {
		return fmt.Errorf("missing user token: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) deleteUser(ctx context.Context, req models.DeleteUserRequest) (models.DeleteUserResponse, error) {
	if err := validateDeleteUserRequest(req); err != nil {
		return models.DeleteUserResponse{}, err
	}

	err := p.client.DeleteUser(ctx, req.OpenBankingForwardedUser.Metadata[UserTokenMetadataKey])
	if err != nil {
		return models.DeleteUserResponse{}, fmt.Errorf("failed to delete user: %w", err)
	}

	return models.DeleteUserResponse{}, nil
}
