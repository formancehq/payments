package plaid

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/pkg/connectors/plaid/client"
	"github.com/formancehq/payments/pkg/connector"
)

func validateCompleteUserLinkRequest(req connector.CompleteUserLinkRequest) error {
	if req.RelatedAttempt == nil {
		return fmt.Errorf("related attempt is required: %w", connector.ErrInvalidRequest)
	}

	if req.RelatedAttempt.TemporaryToken == nil {
		return fmt.Errorf("missing temporary token: %w", connector.ErrInvalidRequest)
	}

	linkToken, ok := req.HTTPCallInformation.QueryValues[client.LinkTokenQueryParamID]
	if !ok || len(linkToken) != 1 {
		return fmt.Errorf("missing link token: %w", connector.ErrInvalidRequest)
	}

	if req.RelatedAttempt.TemporaryToken.Token != linkToken[0] {
		return fmt.Errorf("link token mismatch: %w", connector.ErrInvalidRequest)
	}

	if req.HTTPCallInformation.QueryValues == nil {
		return fmt.Errorf("missing query values: %w", connector.ErrInvalidRequest)
	}

	_, ok = req.HTTPCallInformation.QueryValues[client.PublicTokenQueryParamID]
	if !ok {
		return fmt.Errorf("missing public token: %w", connector.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) completeUserLink(ctx context.Context, req connector.CompleteUserLinkRequest) (connector.CompleteUserLinkResponse, error) {
	if err := validateCompleteUserLinkRequest(req); err != nil {
		return connector.CompleteUserLinkResponse{}, err
	}

	exchangePublicTokenResponse, err := p.client.ExchangePublicToken(ctx, client.ExchangePublicTokenRequest{
		PublicToken: req.HTTPCallInformation.QueryValues[client.PublicTokenQueryParamID][0],
	})
	if err != nil {
		return connector.CompleteUserLinkResponse{}, err
	}

	return connector.CompleteUserLinkResponse{
		Success: &connector.UserLinkSuccessResponse{
			Connections: []connector.PSPOpenBankingConnection{
				{
					CreatedAt:    time.Now().UTC(),
					ConnectionID: exchangePublicTokenResponse.ItemID,
					AccessToken: &connector.Token{
						Token: exchangePublicTokenResponse.AccessToken,
					},
				},
			},
		},
	}, nil
}
