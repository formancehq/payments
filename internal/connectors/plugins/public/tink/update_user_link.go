package tink

import (
	"context"
	"fmt"
	"net/url"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
)

func validateUpdateUserLinkRequest(req models.UpdateUserLinkRequest) error {
	if req.PaymentServiceUser == nil {
		return fmt.Errorf("missing payment service user: %w", models.ErrInvalidRequest)
	}

	if req.Connection == nil {
		return fmt.Errorf("missing connection: %w", models.ErrInvalidRequest)
	}

	if req.Connection.ConnectionID == "" {
		return fmt.Errorf("missing connection ID: %w", models.ErrInvalidRequest)
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

	return nil
}

func (p *Plugin) updateUserLink(ctx context.Context, req models.UpdateUserLinkRequest) (models.UpdateUserLinkResponse, error) {
	if err := validateUpdateUserLinkRequest(req); err != nil {
		return models.UpdateUserLinkResponse{}, err
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
		return models.UpdateUserLinkResponse{}, err
	}

	u, err := url.Parse(fmt.Sprintf("%s/update-consent", tinkLinkBaseURL))
	if err != nil {
		return models.UpdateUserLinkResponse{}, err
	}

	// We have to build the query manually because we don't want to escape the
	// redirect url
	query := url.Values{}
	query.Add("client_id", p.clientID)
	query.Add("state", req.CallBackState)
	query.Add("credentials_id", req.Connection.ConnectionID)
	query.Add("authorization_code", temporaryCodeResponse.Code)
	u.RawQuery = query.Encode()
	// We need to add the redirect URI to the query string directly because
	// the encoded redirect URI is not UI friendly
	u.RawQuery += "&redirect_uri=" + *req.FormanceRedirectURL

	return models.UpdateUserLinkResponse{
		Link: u.String(),
		TemporaryLinkToken: &models.Token{
			Token: temporaryCodeResponse.Code,
			// Tink provides no expiration for this token
		},
	}, nil
}
