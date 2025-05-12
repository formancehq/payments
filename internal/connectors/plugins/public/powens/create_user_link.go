package powens

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createUserLink(ctx context.Context, req models.CreateUserLinkRequest) (models.CreateUserLinkResponse, error) {
	createUserResponse, err := p.client.CreateUser(ctx)
	if err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	temporaryLinkResponse, err := p.client.CreateTemporaryLink(ctx, client.CreateTemporaryLinkRequest{
		AccessToken: createUserResponse.AuthToken,
		RedirectURI: req.RedirectURI,
	})
	if err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	url := fmt.Sprintf("https://webview.powens.com/connect?domain=formance-sandbox&client_id=%s&redirect_uri=%s&code=%s", p.clientID, req.RedirectURI, temporaryLinkResponse.Code)

	return models.CreateUserLinkResponse{
		Link: url,
		TemporaryLinkToken: &models.Token{
			Token:     temporaryLinkResponse.Code,
			ExpiresAt: time.Now().Add(time.Duration(temporaryLinkResponse.ExpiredIn) * time.Second),
		},
		PermanentToken: &models.Token{
			Token: createUserResponse.AuthToken,
			// No expiration for permanent token
		},
	}, nil
}
