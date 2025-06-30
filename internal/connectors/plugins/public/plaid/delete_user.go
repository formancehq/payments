package plaid

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

func validateDeleteUserRequest(req models.DeleteUserRequest) error {
	if req.PSUBankBridge == nil {
		return fmt.Errorf("bank bridge connections are required: %w", models.ErrInvalidRequest)
	}

	if req.PSUBankBridge.Metadata == nil {
		return fmt.Errorf("bank bridge connections metadata are required: %w", models.ErrInvalidRequest)
	}

	if _, ok := req.PSUBankBridge.Metadata[UserTokenMetadataKey]; !ok {
		return fmt.Errorf("missing user token: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) deleteUser(ctx context.Context, req models.DeleteUserRequest) (models.DeleteUserResponse, error) {
	if err := validateDeleteUserRequest(req); err != nil {
		return models.DeleteUserResponse{}, err
	}

	err := p.client.DeleteUser(ctx, req.PSUBankBridge.Metadata[UserTokenMetadataKey])
	if err != nil {
		return models.DeleteUserResponse{}, fmt.Errorf("failed to delete user: %w", err)
	}

	return models.DeleteUserResponse{}, nil
}
