package tink

import (
	"context"
	"fmt"
	"net/url"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
)

var (
	refreshableItems = []string{
		"CHECKING_ACCOUNTS",
		"CHECKING_TRANSACTIONS",
		"SAVING_ACCOUNTS",
		"SAVING_TRANSACTIONS",
		"CREDITCARD_ACCOUNTS",
		"CREDITCARD_TRANSACTIONS",
		"TRANSFER_DESTINATIONS",
	}
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

	u, err := url.Parse("https://link.tink.com/1.0/transactions/connect-accounts")
	if err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	// We have to build the query manually because we don't want to escape the
	// redirect url
	u.RawQuery = fmt.Sprintf(
		"client_id=%s&redirect_uri=%s&state=%s&authorization_code=%s&market=%s&locale=%s&refreshable_items=CHECKING_ACCOUNTS&refreshable_items=CHECKING_TRANSACTIONS&refreshable_items=SAVING_ACCOUNTS&refreshable_items=SAVING_TRANSACTIONS&refreshable_items=CREDITCARD_ACCOUNTS&refreshable_items=CREDITCARD_TRANSACTIONS&refreshable_items=TRANSFER_DESTINATIONS",
		url.QueryEscape(p.clientID),
		*req.FormanceRedirectURL, // Don't escape the redirect url
		url.QueryEscape(req.CallBackState),
		url.QueryEscape(temporaryCodeResponse.Code),
		url.QueryEscape(*req.PaymentServiceUser.Address.Country),
		url.QueryEscape(*req.PaymentServiceUser.ContactDetails.Locale),
	)

	return models.CreateUserLinkResponse{
		Link: u.String(),
		TemporaryLinkToken: &models.Token{
			Token: temporaryCodeResponse.Code,
			// Tink provides no expiration for this token
		},
	}, nil
}
