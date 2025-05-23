package plaid

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
)

func validateCompleteUserLinkRequest(req models.CompleteUserLinkRequest) error {
	if req.RelatedAttempt == nil {
		return fmt.Errorf("related attempt is required: %w", models.ErrInvalidRequest)
	}

	if req.RelatedAttempt.TemporaryToken == nil {
		return fmt.Errorf("missing temporary token: %w", models.ErrInvalidRequest)
	}

	linkToken, ok := req.QueryValues[client.LinkTokenQueryParamID]
	if !ok || len(linkToken) != 1 {
		return fmt.Errorf("missing link token: %w", models.ErrInvalidRequest)
	}

	if *req.RelatedAttempt.TemporaryToken != linkToken[0] {
		return fmt.Errorf("link token mismatch: %w", models.ErrInvalidRequest)
	}

	_, ok = req.QueryValues[client.PublicTokenQueryParamID]
	if !ok {
		return fmt.Errorf("missing public token: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) completeUserLink(ctx context.Context, req models.CompleteUserLinkRequest) (models.CompleteUserLinkResponse, error) {
	if err := validateCompleteUserLinkRequest(req); err != nil {
		return models.CompleteUserLinkResponse{}, err
	}

	exchangePublicTokenResponse, err := p.client.ExchangePublicToken(ctx, client.ExchangePublicTokenRequest{
		PublicToken: req.QueryValues[client.PublicTokenQueryParamID][0],
	})
	if err != nil {
		return models.CompleteUserLinkResponse{}, err
	}

	return models.CompleteUserLinkResponse{
		Success: &models.CompleteUserLinkSuccessResponse{
			Connections: []models.PSUBankBridgeConnection{
				{
					ConnectionID: exchangePublicTokenResponse.ItemID,
					AccessToken:  &exchangePublicTokenResponse.AccessToken,
				},
			},
		},
		Error: &models.CompleteUserLinkErrorResponse{},
	}, nil
}
