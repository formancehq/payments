package tink

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/ce/plugins/tink/client"
	"github.com/formancehq/payments/pkg/domain/models"
)

func validateDeleteUserRequest(req models.DeleteUserRequest) error {
	if req.PaymentServiceUser == nil {
		return fmt.Errorf("paymentServiceUser is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) deleteUser(ctx context.Context, req models.DeleteUserRequest) (models.DeleteUserResponse, error) {
	if err := validateDeleteUserRequest(req); err != nil {
		return models.DeleteUserResponse{}, err
	}

	err := p.client.DeleteUser(ctx, client.DeleteUserRequest{
		UserID: req.PaymentServiceUser.ID.String(),
	})
	if err != nil {
		return models.DeleteUserResponse{}, err
	}

	return models.DeleteUserResponse{}, nil
}
