package tink

import (
	"context"
	"fmt"
	"net/url"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
)

func validateCreateUserLinkRequest(req models.CreateUserLinkRequest) error {
	if req.PaymentServiceUser == nil {
		return fmt.Errorf("missing payment service user: %w", models.ErrInvalidRequest)
	}

	if req.FormanceRedirectURL == nil || *req.FormanceRedirectURL == "" {
		return fmt.Errorf("missing formanceRedirectURL: %w", models.ErrInvalidRequest)
	}

	if req.CallBackState == "" {
		return fmt.Errorf("missing callBackState: %w", models.ErrInvalidRequest)
	}

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

	temporaryCodeResponse, err := p.client.CreateTemporaryAuthorizationCode(ctx, client.CreateTemporaryCodeRequest{
		UserID:   req.PaymentServiceUser.ID.String(),
		Username: req.PaymentServiceUser.Name,
		WantedScopes: []client.Scopes{
			client.SCOPES_AUTHORIZATION_READ,
			client.SCOPES_AUTHORIZATION_GRANT,
			client.SCOPES_CREDENTIALS_REFRESH,
			client.SCOPES_CREDENTIALS_READ,
			client.SCOPES_CREDENTIALS_WRITE,
			client.SCOPES_PROVIDERS_READ,
			client.SCOPES_USER_READ,
		},
	})
	if err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	u, err := url.Parse(fmt.Sprintf("%s/connect-accounts", tinkLinkBaseURL))
	if err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	// We have to build the query manually because we don't want to escape the
	// redirect url
	query := url.Values{}
	query.Add("client_id", p.clientID)
	query.Add("state", req.CallBackState)
	query.Add("authorization_code", temporaryCodeResponse.Code)
	query.Add("market", *req.PaymentServiceUser.Address.Country)
	query.Add("locale", *req.PaymentServiceUser.ContactDetails.Locale)
	u.RawQuery = query.Encode()
	// We need to add the redirect URI to the query string directly because
	// the encoded redirect URI is not UI friendly
	u.RawQuery += "&redirect_uri=" + *req.FormanceRedirectURL

	return models.CreateUserLinkResponse{
		Link: u.String(),
		TemporaryLinkToken: &models.Token{
			Token: temporaryCodeResponse.Code,
			// Tink provides no expiration for this token
		},
	}, nil
}
