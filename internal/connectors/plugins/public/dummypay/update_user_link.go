package dummypay

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/dummypay/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func validateUpdateUserLinkRequest(req models.UpdateUserLinkRequest) error {
	if req.AttemptID == "" {
		return fmt.Errorf("missing attempt ID: %w", models.ErrInvalidRequest)
	}

	if req.FormanceRedirectURL == nil || *req.FormanceRedirectURL == "" {
		return fmt.Errorf("missing formance redirect URI: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) updateUserLink(ctx context.Context, req models.UpdateUserLinkRequest) (models.UpdateUserLinkResponse, error) {
	if err := validateUpdateUserLinkRequest(req); err != nil {
		return models.UpdateUserLinkResponse{}, err
	}

	url, err := url.Parse(*req.FormanceRedirectURL)
	if err != nil {
		return models.UpdateUserLinkResponse{}, fmt.Errorf("failed to parse formance redirect URI: %w", err)
	}

	query := url.Query()
	if p.config.UpdateLinkFlowError {
		query.Set(client.StatusQueryParamID, string(client.LinkStatusError))
	} else {
		query.Set(client.StatusQueryParamID, string(client.LinkStatusSuccess))
	}

	url.RawQuery = query.Encode()

	return models.UpdateUserLinkResponse{
		Link: url.String(),
		TemporaryLinkToken: &models.Token{
			Token:     uuid.New().String(),
			ExpiresAt: time.Now().Add(time.Hour * 24),
		},
	}, nil
}
