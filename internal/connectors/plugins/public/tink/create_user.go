package tink

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

func validateCreateUserRequest(req models.CreateUserRequest) error {
	if req.PaymentServiceUser == nil {
		return fmt.Errorf("payment service user is required: %w", models.ErrInvalidRequest)
	}

	if req.PaymentServiceUser.Address == nil {
		return fmt.Errorf("payment service user address is required: %w", models.ErrInvalidRequest)
	}

	if req.PaymentServiceUser.Address.Country == nil {
		return fmt.Errorf("payment service user address country is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) createUser(ctx context.Context, req models.CreateUserRequest) (models.CreateUserResponse, error) {
	if err := validateCreateUserRequest(req); err != nil {
		return models.CreateUserResponse{}, err
	}

	createUserResponse, err := p.client.CreateUser(ctx,
		req.PaymentServiceUser.ID.String(), *req.PaymentServiceUser.Address.Country)
	if err != nil {
		return models.CreateUserResponse{}, err
	}

	return models.CreateUserResponse{
		PSPUserID: &createUserResponse.UserID,
	}, nil
}
