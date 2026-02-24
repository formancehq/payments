package tink

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/pkg/connector"
)

func validateCreateUserRequest(req connector.CreateUserRequest) error {
	if req.PaymentServiceUser == nil {
		return fmt.Errorf("payment service user is required: %w", connector.ErrInvalidRequest)
	}

	if req.PaymentServiceUser.Address == nil {
		return fmt.Errorf("payment service user address is required: %w", connector.ErrInvalidRequest)
	}

	if req.PaymentServiceUser.Address.Country == nil {
		return fmt.Errorf("payment service user address country is required: %w", connector.ErrInvalidRequest)
	}

	if _, ok := supportedMarkets[*req.PaymentServiceUser.Address.Country]; !ok {
		return fmt.Errorf("unsupported payment service user country: %s: %w", *req.PaymentServiceUser.Address.Country, connector.ErrInvalidRequest)
	}

	if req.PaymentServiceUser.ContactDetails == nil ||
		req.PaymentServiceUser.ContactDetails.Locale == nil ||
		*req.PaymentServiceUser.ContactDetails.Locale == "" {
		return fmt.Errorf("missing payment service user locale: %w", connector.ErrInvalidRequest)
	}

	if _, ok := supportedLocales[*req.PaymentServiceUser.ContactDetails.Locale]; !ok {
		return fmt.Errorf("unsupported payment service user locale: %s: %w", *req.PaymentServiceUser.ContactDetails.Locale, connector.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) createUser(ctx context.Context, req connector.CreateUserRequest) (connector.CreateUserResponse, error) {
	if err := validateCreateUserRequest(req); err != nil {
		return connector.CreateUserResponse{}, err
	}

	createUserResponse, err := p.client.CreateUser(ctx,
		req.PaymentServiceUser.ID.String(), *req.PaymentServiceUser.Address.Country, *req.PaymentServiceUser.ContactDetails.Locale)
	if err != nil {
		return connector.CreateUserResponse{}, err
	}

	return connector.CreateUserResponse{
		PSPUserID: &createUserResponse.UserID,
	}, nil
}
