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

	if req.OpenBankingForwardedUser == nil {
		return fmt.Errorf("open banking connections are required: %w", models.ErrInvalidRequest)
	}

	if req.OpenBankingForwardedUser.AccessToken == nil {
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
		AccessToken: req.OpenBankingForwardedUser.AccessToken.Token,
	})
	if err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	connectURL, err := url.JoinPath(powensWebviewBaseURL, "connect")
	if err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	u, err := url.Parse(connectURL)
	if err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	query := u.Query()
	query.Add("domain", p.config.Domain)
	query.Add("client_id", p.clientID)
	query.Add("code", temporaryCodeResponse.Code)
	query.Add("state", req.CallBackState)
	query.Add("max_connections", strconv.FormatUint(uint64(p.config.MaxConnectionsPerLink), 10))
	u.RawQuery = query.Encode()
	// We need to add the redirect URI to the query string directly because
	// the encoded redirect URI is not UI friendly
	u.RawQuery += "&redirect_uri=" + *req.FormanceRedirectURL

	return models.CreateUserLinkResponse{
		Link: u.String(),
		TemporaryLinkToken: &models.Token{
			Token:     temporaryCodeResponse.Code,
			ExpiresAt: time.Now().Add(time.Duration(temporaryCodeResponse.ExpiresIn) * time.Second),
		},
	}, nil
}
