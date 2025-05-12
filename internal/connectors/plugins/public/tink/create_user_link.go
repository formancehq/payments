package tink

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
)

func validateCreateUserLinkRequest(req models.CreateUserLinkRequest) error {
	if req.PaymentServiceUser.Address == nil || req.PaymentServiceUser.Address.Country == nil {
		return fmt.Errorf("missing payment service user country: %w", models.ErrInvalidRequest)
	}

	if _, ok := supportedMarkets[*req.PaymentServiceUser.Address.Country]; !ok {
		return fmt.Errorf("unsupported payment service user country: %s: %w", *req.PaymentServiceUser.Address.Country, models.ErrInvalidRequest)
	}

	if req.PaymentServiceUser.ContactDetails == nil ||
		req.PaymentServiceUser.ContactDetails.Locale == nil ||
		*req.PaymentServiceUser.ContactDetails.Locale == "" {
		return fmt.Errorf("missing payment service user locale: %w", models.ErrInvalidRequest)
	}

	if _, ok := supportedLocales[*req.PaymentServiceUser.ContactDetails.Locale]; !ok {
		return fmt.Errorf("unsupported payment service user locale: %s: %w", *req.PaymentServiceUser.ContactDetails.Locale, models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) createUserLink(ctx context.Context, req models.CreateUserLinkRequest) (models.CreateUserLinkResponse, error) {
	if err := validateCreateUserLinkRequest(req); err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	createUserResponse, err := p.client.CreateUser(ctx, req.PaymentServiceUser.ID.String(), *req.PaymentServiceUser.Address.Country)
	if err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	temporaryCodeResponse, err := p.client.CreateTemporaryCode(ctx, client.CreateTemporaryCodeRequest{
		UserID:   createUserResponse.ExternalUserID,
		Username: req.PaymentServiceUser.Name,
	})
	if err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	// TODO(polo): allow the user to choose which link to have (between business transactions and personal transactions)
	url := fmt.Sprintf("https://link.tink.com/1.0/transactions/connect-accounts?client_id=%s&redirect_uri=%s&authorization_code=%s&market=%s&locale=%s",
		p.clientID, req.RedirectURI, temporaryCodeResponse.Code, *req.PaymentServiceUser.Address.Country, *req.PaymentServiceUser.ContactDetails.Locale)

	return models.CreateUserLinkResponse{
		Link: url,
		TemporaryLinkToken: &models.Token{
			Token: temporaryCodeResponse.Code,
			// Tink provides no expiration for this token
		},
	}, nil
}
