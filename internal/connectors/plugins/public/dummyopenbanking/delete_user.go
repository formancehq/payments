package dummyopenbanking

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

func validateDeleteUserRequest(req models.DeleteUserRequest) error {
	if req.PaymentServiceUser == nil {
		return fmt.Errorf("payment service user is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) deleteUser(ctx context.Context, req models.DeleteUserRequest) (models.DeleteUserResponse, error) {
	if err := validateDeleteUserRequest(req); err != nil {
		return models.DeleteUserResponse{}, err
	}

	err := p.client.DeleteUser(ctx, req.PaymentServiceUser.ID.String())
	if err != nil {
		return models.DeleteUserResponse{}, fmt.Errorf("failed to delete user: %w", err)
	}

	return models.DeleteUserResponse{}, nil
}
