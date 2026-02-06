package tink

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/pkg/connectors/tink/client"
	"github.com/formancehq/payments/pkg/connector"
)

func validateDeleteUserConnectionRequest(req connector.DeleteUserConnectionRequest) error {
	if req.PaymentServiceUser == nil {
		return fmt.Errorf("paymentServiceUser is required: %w", connector.ErrInvalidRequest)
	}

	if req.PaymentServiceUser.Name == "" {
		return fmt.Errorf("name is required: %w", connector.ErrInvalidRequest)
	}

	if req.Connection == nil {
		return fmt.Errorf("connection is required: %w", connector.ErrInvalidRequest)
	}

	if req.Connection.ConnectionID == "" {
		return fmt.Errorf("connectionID is required: %w", connector.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) deleteUserConnection(ctx context.Context, req connector.DeleteUserConnectionRequest) (connector.DeleteUserConnectionResponse, error) {
	if err := validateDeleteUserConnectionRequest(req); err != nil {
		return connector.DeleteUserConnectionResponse{}, err
	}

	err := p.client.DeleteUserConnection(ctx, client.DeleteUserConnectionRequest{
		UserID:        req.PaymentServiceUser.ID.String(),
		Username:      req.PaymentServiceUser.Name,
		CredentialsID: req.Connection.ConnectionID,
	})
	if err != nil {
		return connector.DeleteUserConnectionResponse{}, err
	}

	return connector.DeleteUserConnectionResponse{}, nil
}
