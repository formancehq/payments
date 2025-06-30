package plaid

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
)

func validateUpdateUserLinkRequest(req models.UpdateUserLinkRequest) error {
	if req.Connection == nil {
		return fmt.Errorf("missing connection: %w", models.ErrInvalidRequest)
	}

	if req.Connection.AccessToken == nil {
		return fmt.Errorf("missing access token: %w", models.ErrInvalidRequest)
	}

	if req.PaymentServiceUser == nil {
		return fmt.Errorf("missing payment service user: %w", models.ErrInvalidRequest)
	}

	if req.PaymentServiceUser.Name == "" {
		return fmt.Errorf("missing payment service user name: %w", models.ErrInvalidRequest)
	}

	if req.PaymentServiceUser.ContactDetails == nil ||
		req.PaymentServiceUser.ContactDetails.Locale == nil ||
		*req.PaymentServiceUser.ContactDetails.Locale == "" {
		return fmt.Errorf("missing payment service user locale: %w", models.ErrInvalidRequest)
	}

	if req.PaymentServiceUser.Address == nil || req.PaymentServiceUser.Address.Country == nil {
		return fmt.Errorf("missing payment service user country: %w", models.ErrInvalidRequest)
	}

	if _, ok := supportedCountryCodes[*req.PaymentServiceUser.Address.Country]; !ok {
		return fmt.Errorf("unsupported payment service user country: %s: %w", *req.PaymentServiceUser.Address.Country, models.ErrInvalidRequest)
	}

	if req.ClientRedirectURL == nil || *req.ClientRedirectURL == "" {
		return fmt.Errorf("missing redirect URI: %w", models.ErrInvalidRequest)
	}

	if req.PSUBankBridge == nil {
		return fmt.Errorf("missing bank bridge connections: %w", models.ErrInvalidRequest)
	}

	if req.PSUBankBridge.Metadata == nil {
		return fmt.Errorf("missing bank bridge connections metadata: %w", models.ErrInvalidRequest)
	}

	if _, ok := req.PSUBankBridge.Metadata[UserTokenMetadataKey]; !ok {
		return fmt.Errorf("missing user token: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) updateUserLink(ctx context.Context, req models.UpdateUserLinkRequest) (models.UpdateUserLinkResponse, error) {
	if err := validateUpdateUserLinkRequest(req); err != nil {
		return models.UpdateUserLinkResponse{}, err
	}

	language, err := validateLanguageCode(*req.PaymentServiceUser.ContactDetails.Locale)
	if err != nil {
		return models.UpdateUserLinkResponse{}, err
	}

	resp, err := p.client.UpdateLinkToken(ctx, client.UpdateLinkTokenRequest{
		UserName:    req.PaymentServiceUser.Name,
		UserID:      req.PaymentServiceUser.ID.String(),
		UserToken:   req.PSUBankBridge.Metadata[UserTokenMetadataKey],
		Language:    language,
		CountryCode: *req.PaymentServiceUser.Address.Country,
		RedirectURI: *req.ClientRedirectURL,
		AccessToken: req.Connection.AccessToken.Token,
		ItemID:      req.Connection.ConnectionID,
	})
	if err != nil {
		return models.UpdateUserLinkResponse{}, err
	}

	return models.UpdateUserLinkResponse{
		Link: resp.HostedLinkUrl,
		TemporaryLinkToken: &models.Token{
			Token:     resp.LinkToken,
			ExpiresAt: resp.Expiration,
		},
	}, nil
}
