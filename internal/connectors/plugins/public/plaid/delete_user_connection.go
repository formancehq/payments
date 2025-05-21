package plaid

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
)

func validateDeleteUserConnectionRequest(req models.DeleteUserConnectionRequest) error {
	if req.PaymentServiceUser == nil {
		return fmt.Errorf("payment service user is required: %w", models.ErrInvalidRequest)
	}

	if req.BankBridgeConsent == nil {
		return fmt.Errorf("bank bridge consent is required: %w", models.ErrInvalidRequest)
	}

	if req.BankBridgeConsent.AccessToken == "" {
		return fmt.Errorf("access token is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) deleteUserConnection(ctx context.Context, req models.DeleteUserConnectionRequest) (models.DeleteUserConnectionResponse, error) {
	if err := validateDeleteUserConnectionRequest(req); err != nil {
		return models.DeleteUserConnectionResponse{}, err
	}

	err := p.client.DeleteItem(ctx, client.DeleteItemRequest{
		AccessToken: req.BankBridgeConsent.AccessToken,
	})
	if err != nil {
		return models.DeleteUserConnectionResponse{}, fmt.Errorf("failed to delete item: %w", err)
	}

	return models.DeleteUserConnectionResponse{}, nil
}
