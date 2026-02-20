package tink

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/pkg/connectors/tink/client"
	"github.com/formancehq/payments/pkg/connector"
)

func validateDeleteUserRequest(req connector.DeleteUserRequest) error {
	if req.PaymentServiceUser == nil {
		return fmt.Errorf("paymentServiceUser is required: %w", connector.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) deleteUser(ctx context.Context, req connector.DeleteUserRequest) (connector.DeleteUserResponse, error) {
	if err := validateDeleteUserRequest(req); err != nil {
		return connector.DeleteUserResponse{}, err
	}

	err := p.client.DeleteUser(ctx, client.DeleteUserRequest{
		UserID: req.PaymentServiceUser.ID.String(),
	})
	if err != nil {
		return connector.DeleteUserResponse{}, err
	}

	return connector.DeleteUserResponse{}, nil
}
