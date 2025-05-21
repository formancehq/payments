package powens

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
)

func validateDeleteUserConnectionRequest(req models.DeleteUserConnectionRequest) error {
	if req.BankBridgeConsent == nil {
		return fmt.Errorf("bank bridge consent is required: %w", models.ErrInvalidRequest)
	}

	if req.BankBridgeConsent.AccessToken == "" {
		return fmt.Errorf("access token is required: %w", models.ErrInvalidRequest)
	}

	if req.Connection == nil {
		return fmt.Errorf("connection is required: %w", models.ErrInvalidRequest)
	}

	if req.Connection.ConnectionID == "" {
		return fmt.Errorf("connection id is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) deleteUserConnection(ctx context.Context, req models.DeleteUserConnectionRequest) (models.DeleteUserConnectionResponse, error) {
	if err := validateDeleteUserConnectionRequest(req); err != nil {
		return models.DeleteUserConnectionResponse{}, err
	}

	err := p.client.DeleteUserConnection(ctx, client.DeleteUserConnectionRequest{
		AccessToken:  req.BankBridgeConsent.AccessToken,
		ConnectionID: req.Connection.ConnectionID,
	})
	if err != nil {
		return models.DeleteUserConnectionResponse{}, err
	}

	return models.DeleteUserConnectionResponse{}, nil
}
