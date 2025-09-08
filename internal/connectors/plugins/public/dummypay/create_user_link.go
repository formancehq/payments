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

func validateCreateUserLinkRequest(req models.CreateUserLinkRequest) error {
	if req.AttemptID == "" {
		return fmt.Errorf("missing attempt ID: %w", models.ErrInvalidRequest)
	}

	if req.FormanceRedirectURL == nil || *req.FormanceRedirectURL == "" {
		return fmt.Errorf("missing formance redirect URI: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) createUserLink(_ context.Context, req models.CreateUserLinkRequest) (models.CreateUserLinkResponse, error) {
	if err := validateCreateUserLinkRequest(req); err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	url, err := url.Parse(*req.FormanceRedirectURL)
	if err != nil {
		return models.CreateUserLinkResponse{}, fmt.Errorf("failed to parse formance redirect URI: %w", err)
	}

	query := url.Query()
	query.Set(models.NoRedirectQueryParamID, "true")
	query.Set(models.StateQueryParamID, req.CallBackState)
	query.Set(client.UserIDQueryParamID, req.PaymentServiceUser.ID.String())
	if p.config.CreateLinkFlowError {
		query.Set(client.StatusQueryParamID, string(client.LinkStatusError))
	} else {
		query.Set(client.StatusQueryParamID, string(client.LinkStatusSuccess))
	}

	url.RawQuery = query.Encode()

	return models.CreateUserLinkResponse{
		Link: url.String(),
		TemporaryLinkToken: &models.Token{
			Token:     uuid.New().String(),
			ExpiresAt: time.Now().Add(time.Hour * 24),
		},
	}, nil
}
