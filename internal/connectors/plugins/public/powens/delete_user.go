package powens

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
)

func validateDeleteUserRequest(req models.DeleteUserRequest) error {
	if req.PaymentServiceUser == nil {
		return fmt.Errorf("payment service user is required: %w", models.ErrInvalidRequest)
	}

	if req.PaymentServiceUser.BankBridgeConnections == nil {
		return fmt.Errorf("bank bridge connections are required: %w", models.ErrInvalidRequest)
	}

	if req.PaymentServiceUser.BankBridgeConnections.AccessToken == nil {
		return fmt.Errorf("auth token is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) deleteUser(ctx context.Context, req models.DeleteUserRequest) (models.DeleteUserResponse, error) {
	if err := validateDeleteUserRequest(req); err != nil {
		return models.DeleteUserResponse{}, err
	}

	err := p.client.DeleteUser(ctx, client.DeleteUserRequest{
		AccessToken: req.PaymentServiceUser.BankBridgeConnections.AccessToken.Token,
	})
	if err != nil {
		return models.DeleteUserResponse{}, err
	}

	return models.DeleteUserResponse{}, nil
}
