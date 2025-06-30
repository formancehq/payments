package plaid

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

	userToken, err := p.client.CreateUser(ctx, req.PaymentServiceUser.ID.String())
	if err != nil {
		return models.CreateUserResponse{}, err
	}

	return models.CreateUserResponse{
		Metadata: map[string]string{
			UserTokenMetadataKey: userToken,
		},
	}, nil
}
