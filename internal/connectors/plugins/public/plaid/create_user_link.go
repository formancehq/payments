package plaid

import (
	"context"
	"fmt"

	iso6391 "github.com/emvi/iso-639-1"
	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
	"golang.org/x/text/language"
)

func validateCreateUserLinkRequest(req models.CreateUserLinkRequest) error {
	if req.ApplicationName == "" {
		return fmt.Errorf("missing application name: %w", models.ErrInvalidRequest)
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

	if req.OpenBankingForwardedUser == nil {
		return fmt.Errorf("missing open banking connections: %w", models.ErrInvalidRequest)
	}

	if req.OpenBankingForwardedUser.Metadata == nil {
		return fmt.Errorf("missing open banking connections metadata: %w", models.ErrInvalidRequest)
	}

	if _, ok := req.OpenBankingForwardedUser.Metadata[UserTokenMetadataKey]; !ok {
		return fmt.Errorf("missing user token: %w", models.ErrInvalidRequest)
	}

	return nil
}

func validateLanguageCode(locale string) (string, error) {
	l, err := language.Parse(locale)
	if err != nil {
		return "", fmt.Errorf("invalid locale: %s: %w: %w", locale, err, models.ErrInvalidRequest)
	}

	iso639LanguageCode, _ := l.Base()
	if !iso6391.ValidCode(iso639LanguageCode.String()) {
		return "", fmt.Errorf("locale base needs to be in iso639-1 format: %s: %w", locale, models.ErrInvalidRequest)
	}

	if _, ok := supportedLanguage[iso639LanguageCode.String()]; !ok {
		return "", fmt.Errorf("unsupported locale: %s: %w", iso639LanguageCode.String(), models.ErrInvalidRequest)
	}

	return iso639LanguageCode.String(), nil
}

func (p *Plugin) createUserLink(ctx context.Context, req models.CreateUserLinkRequest) (models.CreateUserLinkResponse, error) {
	if err := validateCreateUserLinkRequest(req); err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	language, err := validateLanguageCode(*req.PaymentServiceUser.ContactDetails.Locale)
	if err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	resp, err := p.client.CreateLinkToken(ctx, client.CreateLinkTokenRequest{
		ApplicationName: req.ApplicationName,
		UserID:          req.PaymentServiceUser.ID.String(),
		UserToken:       req.OpenBankingForwardedUser.Metadata[UserTokenMetadataKey],
		Language:        language,
		CountryCode:     *req.PaymentServiceUser.Address.Country,
		RedirectURI:     *req.ClientRedirectURL,
		WebhookBaseURL:  req.WebhookBaseURL,
		AttemptID:       req.AttemptID,
	})
	if err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	return models.CreateUserLinkResponse{
		Link: resp.HostedLinkUrl,
		TemporaryLinkToken: &models.Token{
			Token:     resp.LinkToken,
			ExpiresAt: resp.Expiration,
		},
	}, nil
}
