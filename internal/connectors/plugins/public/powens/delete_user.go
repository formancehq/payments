package powens

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
)

func validateDeleteUserRequest(req models.DeleteUserRequest) error {
	if req.BankBridgeConsent == nil {
		return fmt.Errorf("bank bridge consent is required: %w", models.ErrInvalidRequest)
	}

	if req.BankBridgeConsent.AccessToken == "" {
		return fmt.Errorf("access token is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) deleteUser(ctx context.Context, req models.DeleteUserRequest) (models.DeleteUserResponse, error) {
	if err := validateDeleteUserRequest(req); err != nil {
		return models.DeleteUserResponse{}, err
	}

	err := p.client.DeleteUser(ctx, client.DeleteUserRequest{
		AccessToken: req.BankBridgeConsent.AccessToken,
	})
	if err != nil {
		return models.DeleteUserResponse{}, err
	}

	return models.DeleteUserResponse{}, nil
}
