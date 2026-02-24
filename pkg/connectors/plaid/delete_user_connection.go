package plaid

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/pkg/connectors/plaid/client"
	"github.com/formancehq/payments/pkg/connector"
)

func validateDeleteUserConnectionRequest(req connector.DeleteUserConnectionRequest) error {
	if req.Connection == nil {
		return fmt.Errorf("connection is required: %w", connector.ErrInvalidRequest)
	}

	if req.Connection.AccessToken == nil {
		return fmt.Errorf("access token is required: %w", connector.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) deleteUserConnection(ctx context.Context, req connector.DeleteUserConnectionRequest) (connector.DeleteUserConnectionResponse, error) {
	if err := validateDeleteUserConnectionRequest(req); err != nil {
		return connector.DeleteUserConnectionResponse{}, err
	}

	err := p.client.DeleteItem(ctx, client.DeleteItemRequest{
		AccessToken: req.Connection.AccessToken.Token,
	})
	if err != nil {
		return connector.DeleteUserConnectionResponse{}, fmt.Errorf("failed to delete item: %w", err)
	}

	return connector.DeleteUserConnectionResponse{}, nil
}
