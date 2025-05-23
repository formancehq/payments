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

	if req.PaymentServiceUser.BankBridgeConnections == nil {
		return fmt.Errorf("bank bridge connections are required: %w", models.ErrInvalidRequest)
	}

	if req.PaymentServiceUser.BankBridgeConnections.AccessToken == nil {
		return fmt.Errorf("auth token is required: %w", models.ErrInvalidRequest)
	}

	if req.CallBackState == "" {
		return fmt.Errorf("callBackState is required: %w", models.ErrInvalidRequest)
	}

	if req.FormanceRedirectURI == nil || *req.FormanceRedirectURI == "" {
		return fmt.Errorf("formanceRedirectURI is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) createUserLink(ctx context.Context, req models.CreateUserLinkRequest) (models.CreateUserLinkResponse, error) {
	if err := validateCreateUserLinkRequest(req); err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	temporaryLinkResponse, err := p.client.CreateTemporaryLink(ctx, client.CreateTemporaryLinkRequest{
		AccessToken: req.PaymentServiceUser.BankBridgeConnections.AccessToken.Token,
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
	query.Add("redirect_uri", *req.FormanceRedirectURI)
	query.Add("code", temporaryLinkResponse.Code)
	query.Add("state", req.CallBackState)
	query.Add("max_connections", strconv.Itoa(p.config.MaxConnections))
	u.RawQuery = query.Encode()

	return models.CreateUserLinkResponse{
		Link: u.String(),
		TemporaryLinkToken: &models.Token{
			Token:     temporaryLinkResponse.Code,
			ExpiresAt: time.Now().Add(time.Duration(temporaryLinkResponse.ExpiredIn) * time.Second),
		},
	}, nil
}
