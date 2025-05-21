package tink

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
)

func validateDeleteUserConnectionRequest(req models.DeleteUserConnectionRequest) error {
	if req.PaymentServiceUser == nil {
		return fmt.Errorf("paymentServiceUser is required: %w", models.ErrInvalidRequest)
	}

	if req.PaymentServiceUser.Name == "" {
		return fmt.Errorf("name is required: %w", models.ErrInvalidRequest)
	}

	if req.BankBridgeConsent == nil {
		return fmt.Errorf("bankBridgeConsent is required: %w", models.ErrInvalidRequest)
	}

	if req.BankBridgeConsent.AccessToken == "" {
		return fmt.Errorf("accessToken is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) deleteUserConnection(ctx context.Context, req models.DeleteUserConnectionRequest) (models.DeleteUserConnectionResponse, error) {
	if err := validateDeleteUserConnectionRequest(req); err != nil {
		return models.DeleteUserConnectionResponse{}, err
	}

	err := p.client.DeleteUserConnection(ctx, client.DeleteUserConnectionRequest{
		UserID:        req.PaymentServiceUser.ID.String(),
		Username:      req.PaymentServiceUser.Name,
		CredentialsID: req.BankBridgeConsent.AccessToken,
	})
	if err != nil {
		return models.DeleteUserConnectionResponse{}, err
	}

	return models.DeleteUserConnectionResponse{}, nil
}
