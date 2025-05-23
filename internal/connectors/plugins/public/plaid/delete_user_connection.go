package plaid

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
)

func validateDeleteUserConnectionRequest(req models.DeleteUserConnectionRequest) error {
	if req.Connection == nil {
		return fmt.Errorf("connection is required: %w", models.ErrInvalidRequest)
	}

	if req.Connection.AccessToken == nil {
		return fmt.Errorf("access token is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) deleteUserConnection(ctx context.Context, req models.DeleteUserConnectionRequest) (models.DeleteUserConnectionResponse, error) {
	if err := validateDeleteUserConnectionRequest(req); err != nil {
		return models.DeleteUserConnectionResponse{}, err
	}

	err := p.client.DeleteItem(ctx, client.DeleteItemRequest{
		AccessToken: req.Connection.AccessToken.Token,
	})
	if err != nil {
		return models.DeleteUserConnectionResponse{}, fmt.Errorf("failed to delete item: %w", err)
	}

	return models.DeleteUserConnectionResponse{}, nil
}
