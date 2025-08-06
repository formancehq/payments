package powens

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
)

func validateCreateUserLinkRequest(req models.CreateUserLinkRequest) error {
	if req.PaymentServiceUser == nil {
		return fmt.Errorf("payment service user is required: %w", models.ErrInvalidRequest)
	}

	if req.PSUBankBridge == nil {
		return fmt.Errorf("bank bridge connections are required: %w", models.ErrInvalidRequest)
	}

	if req.PSUBankBridge.AccessToken == nil {
		return fmt.Errorf("auth token is required: %w", models.ErrInvalidRequest)
	}

	if req.CallBackState == "" {
		return fmt.Errorf("callBackState is required: %w", models.ErrInvalidRequest)
	}

	if req.FormanceRedirectURL == nil || *req.FormanceRedirectURL == "" {
		return fmt.Errorf("formanceRedirectURL is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) createUserLink(ctx context.Context, req models.CreateUserLinkRequest) (models.CreateUserLinkResponse, error) {
	if err := validateCreateUserLinkRequest(req); err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	temporaryCodeResponse, err := p.client.CreateTemporaryCode(ctx, client.CreateTemporaryLinkRequest{
		AccessToken: req.PSUBankBridge.AccessToken.Token,
	})
	if err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	u, err := url.Parse("https://webview.powens.com/connect")
	if err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	query := u.Query()
	query.Add("domain", p.config.Domain)
	query.Add("client_id", p.clientID)
	query.Add("code", temporaryCodeResponse.Code)
	query.Add("state", req.CallBackState)
	query.Add("max_connections", strconv.FormatUint(uint64(p.config.MaxConnections), 10))
	u.RawQuery = query.Encode()
	u.RawQuery += "&redirect_uri=" + *req.FormanceRedirectURL

	return models.CreateUserLinkResponse{
		Link: u.String(),
		TemporaryLinkToken: &models.Token{
			Token:     temporaryCodeResponse.Code,
			ExpiresAt: time.Now().Add(time.Duration(temporaryCodeResponse.ExpiredIn) * time.Second),
		},
	}, nil
}
