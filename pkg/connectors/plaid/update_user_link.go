package plaid

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/pkg/connectors/plaid/client"
	"github.com/formancehq/payments/pkg/connector"
)

func validateUpdateUserLinkRequest(req connector.UpdateUserLinkRequest) error {
	if req.ApplicationName == "" {
		return fmt.Errorf("missing application name: %w", connector.ErrInvalidRequest)
	}

	if req.Connection == nil {
		return fmt.Errorf("missing connection: %w", connector.ErrInvalidRequest)
	}

	if req.Connection.AccessToken == nil {
		return fmt.Errorf("missing access token: %w", connector.ErrInvalidRequest)
	}

	if req.PaymentServiceUser == nil {
		return fmt.Errorf("missing payment service user: %w", connector.ErrInvalidRequest)
	}

	if req.PaymentServiceUser.ContactDetails == nil ||
		req.PaymentServiceUser.ContactDetails.Locale == nil ||
		*req.PaymentServiceUser.ContactDetails.Locale == "" {
		return fmt.Errorf("missing payment service user locale: %w", connector.ErrInvalidRequest)
	}

	if req.PaymentServiceUser.Address == nil || req.PaymentServiceUser.Address.Country == nil {
		return fmt.Errorf("missing payment service user country: %w", connector.ErrInvalidRequest)
	}

	if _, ok := supportedCountryCodes[*req.PaymentServiceUser.Address.Country]; !ok {
		return fmt.Errorf("unsupported payment service user country: %s: %w", *req.PaymentServiceUser.Address.Country, connector.ErrInvalidRequest)
	}

	if req.ClientRedirectURL == nil || *req.ClientRedirectURL == "" {
		return fmt.Errorf("missing redirect URI: %w", connector.ErrInvalidRequest)
	}

	if req.OpenBankingForwardedUser == nil {
		return fmt.Errorf("missing open banking connections: %w", connector.ErrInvalidRequest)
	}

	if req.OpenBankingForwardedUser.Metadata == nil {
		return fmt.Errorf("missing open banking connections metadata: %w", connector.ErrInvalidRequest)
	}

	if _, ok := req.OpenBankingForwardedUser.Metadata[UserTokenMetadataKey]; !ok {
		return fmt.Errorf("missing user token: %w", connector.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) updateUserLink(ctx context.Context, req connector.UpdateUserLinkRequest) (connector.UpdateUserLinkResponse, error) {
	if err := validateUpdateUserLinkRequest(req); err != nil {
		return connector.UpdateUserLinkResponse{}, err
	}

	language, err := validateLanguageCode(*req.PaymentServiceUser.ContactDetails.Locale)
	if err != nil {
		return connector.UpdateUserLinkResponse{}, err
	}

	resp, err := p.client.UpdateLinkToken(ctx, client.UpdateLinkTokenRequest{
		AttemptID:       req.AttemptID,
		ApplicationName: req.ApplicationName,
		UserID:          req.PaymentServiceUser.ID.String(),
		UserToken:       req.OpenBankingForwardedUser.Metadata[UserTokenMetadataKey],
		Language:        language,
		CountryCode:     *req.PaymentServiceUser.Address.Country,
		RedirectURI:     *req.ClientRedirectURL,
		AccessToken:     req.Connection.AccessToken.Token,
		ItemID:          req.Connection.ConnectionID,
		WebhookBaseURL:  req.WebhookBaseURL,
	})
	if err != nil {
		return connector.UpdateUserLinkResponse{}, err
	}

	return connector.UpdateUserLinkResponse{
		Link: resp.HostedLinkUrl,
		TemporaryLinkToken: &connector.Token{
			Token:     resp.LinkToken,
			ExpiresAt: resp.Expiration,
		},
	}, nil
}
