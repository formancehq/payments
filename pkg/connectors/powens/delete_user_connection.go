package powens

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/pkg/connectors/powens/client"
	"github.com/formancehq/payments/pkg/connector"
)

func validateDeleteUserConnectionRequest(req connector.DeleteUserConnectionRequest) error {
	if req.Connection == nil {
		return fmt.Errorf("connection is required: %w", connector.ErrInvalidRequest)
	}

	if req.Connection.ConnectionID == "" {
		return fmt.Errorf("connection id is required: %w", connector.ErrInvalidRequest)
	}

	if req.OpenBankingForwardedUser == nil {
		return fmt.Errorf("open banking forwarded user is required: %w", connector.ErrInvalidRequest)
	}

	if req.OpenBankingForwardedUser.AccessToken == nil {
		return fmt.Errorf("auth token is required: %w", connector.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) deleteUserConnection(ctx context.Context, req connector.DeleteUserConnectionRequest) (connector.DeleteUserConnectionResponse, error) {
	if err := validateDeleteUserConnectionRequest(req); err != nil {
		return connector.DeleteUserConnectionResponse{}, err
	}

	err := p.client.DeleteUserConnection(ctx, client.DeleteUserConnectionRequest{
		AccessToken:  req.OpenBankingForwardedUser.AccessToken.Token,
		ConnectionID: req.Connection.ConnectionID,
	})
	if err != nil {
		return connector.DeleteUserConnectionResponse{}, err
	}

	return connector.DeleteUserConnectionResponse{}, nil
}
