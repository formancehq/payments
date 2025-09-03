package powens

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
)

func validateDeleteUserRequest(req models.DeleteUserRequest) error {
	if req.OpenBankingProviderPSU == nil {
		return fmt.Errorf("open banking provider psu is required: %w", models.ErrInvalidRequest)
	}

	if req.OpenBankingProviderPSU.AccessToken == nil {
		return fmt.Errorf("auth token is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) deleteUser(ctx context.Context, req models.DeleteUserRequest) (models.DeleteUserResponse, error) {
	if err := validateDeleteUserRequest(req); err != nil {
		return models.DeleteUserResponse{}, err
	}

	err := p.client.DeleteUser(ctx, client.DeleteUserRequest{
		AccessToken: req.OpenBankingProviderPSU.AccessToken.Token,
	})
	if err != nil {
		return models.DeleteUserResponse{}, err
	}

	return models.DeleteUserResponse{}, nil
}
