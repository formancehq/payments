package plaid

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/pkg/connector"
)

func validateCreateUserRequest(req connector.CreateUserRequest) error {
	if req.PaymentServiceUser == nil {
		return fmt.Errorf("payment service user is required: %w", connector.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) createUser(ctx context.Context, req connector.CreateUserRequest) (connector.CreateUserResponse, error) {
	if err := validateCreateUserRequest(req); err != nil {
		return connector.CreateUserResponse{}, err
	}

	userToken, err := p.client.CreateUser(ctx, req.PaymentServiceUser.ID.String())
	if err != nil {
		return connector.CreateUserResponse{}, err
	}

	return connector.CreateUserResponse{
		Metadata: map[string]string{
			UserTokenMetadataKey: userToken,
		},
	}, nil
}
