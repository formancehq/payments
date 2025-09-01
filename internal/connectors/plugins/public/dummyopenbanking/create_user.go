package dummyopenbanking

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

func validateCreateUserRequest(req models.CreateUserRequest) error {
	if req.PaymentServiceUser == nil {
		return fmt.Errorf("payment service user is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) createUser(ctx context.Context, req models.CreateUserRequest) (models.CreateUserResponse, error) {
	if err := validateCreateUserRequest(req); err != nil {
		return models.CreateUserResponse{}, err
	}

	userID, err := p.client.CreateUser(ctx, *req.PaymentServiceUser)
	if err != nil {
		return models.CreateUserResponse{}, err
	}

	return models.CreateUserResponse{
		PSPUserID: &userID,
	}, nil
}
