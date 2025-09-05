package dummypay

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

func validateDeleteUserConnectionRequest(req models.DeleteUserConnectionRequest) error {
	if req.PaymentServiceUser == nil {
		return fmt.Errorf("payment service user is required: %w", models.ErrInvalidRequest)
	}

	if req.Connection == nil {
		return fmt.Errorf("connection is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) deleteUserConnection(ctx context.Context, req models.DeleteUserConnectionRequest) (models.DeleteUserConnectionResponse, error) {
	if err := validateDeleteUserConnectionRequest(req); err != nil {
		return models.DeleteUserConnectionResponse{}, err
	}

	err := p.client.DeleteUserConnection(ctx, req.PaymentServiceUser.ID.String(), req.Connection.ConnectionID)
	if err != nil {
		return models.DeleteUserConnectionResponse{}, fmt.Errorf("failed to delete user connection: %w", err)
	}

	return models.DeleteUserConnectionResponse{}, nil
}
