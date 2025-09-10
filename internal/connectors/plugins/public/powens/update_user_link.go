package powens

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
)

func validateUpdateUserLinkRequest(req models.UpdateUserLinkRequest) error {
	if req.PaymentServiceUser == nil {
		return fmt.Errorf("payment service user is required: %w", models.ErrInvalidRequest)
	}

	if req.OpenBankingForwardedUser == nil {
		return fmt.Errorf("open banking connections are required: %w", models.ErrInvalidRequest)
	}

	if req.OpenBankingForwardedUser.AccessToken == nil {
		return fmt.Errorf("auth token is required: %w", models.ErrInvalidRequest)
	}

	if req.Connection == nil {
		return fmt.Errorf("connection is required: %w", models.ErrInvalidRequest)
	}

	if req.Connection.ConnectionID == "" {
		return fmt.Errorf("connection ID is required: %w", models.ErrInvalidRequest)
	}

	if req.CallBackState == "" {
		return fmt.Errorf("callBackState is required: %w", models.ErrInvalidRequest)
	}

	if req.FormanceRedirectURL == nil || *req.FormanceRedirectURL == "" {
		return fmt.Errorf("formanceRedirectURL is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) updateUserLink(ctx context.Context, req models.UpdateUserLinkRequest) (models.UpdateUserLinkResponse, error) {
	if err := validateUpdateUserLinkRequest(req); err != nil {
		return models.UpdateUserLinkResponse{}, err
	}

	temporaryCodeResponse, err := p.client.CreateTemporaryCode(ctx, client.CreateTemporaryLinkRequest{
		AccessToken: req.OpenBankingForwardedUser.AccessToken.Token,
	})
	if err != nil {
		return models.UpdateUserLinkResponse{}, err
	}

	reconnectURL, err := url.JoinPath(powensWebviewBaseURL, "reconnect")
	if err != nil {
		return models.UpdateUserLinkResponse{}, err
	}

	u, err := url.Parse(reconnectURL)
	if err != nil {
		return models.UpdateUserLinkResponse{}, err
	}

	query := u.Query()
	query.Add("domain", p.config.Domain)
	query.Add("client_id", p.clientID)
	query.Add("code", temporaryCodeResponse.Code)
	query.Add("connection_id", req.Connection.ConnectionID)
	query.Add("state", req.CallBackState)
	u.RawQuery = query.Encode()
	u.RawQuery += "&redirect_uri=" + *req.FormanceRedirectURL

	return models.UpdateUserLinkResponse{
		Link: u.String(),
		TemporaryLinkToken: &models.Token{
			Token:     temporaryCodeResponse.Code,
			ExpiresAt: time.Now().Add(time.Duration(temporaryCodeResponse.ExpiresIn) * time.Second),
		},
	}, nil
}
