package dummyopenbanking

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/dummyopenbanking/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func validateCompleteUserLinkRequest(req models.CompleteUserLinkRequest) error {
	if req.RelatedAttempt == nil {
		return fmt.Errorf("related attempt is required: %w", models.ErrInvalidRequest)
	}

	if req.RelatedAttempt.TemporaryToken == nil {
		return fmt.Errorf("missing temporary token: %w", models.ErrInvalidRequest)
	}

	if req.HTTPCallInformation.QueryValues == nil {
		return fmt.Errorf("missing query values: %w", models.ErrInvalidRequest)
	}

	if req.HTTPCallInformation.QueryValues[client.StatusQueryParamID] == nil ||
		len(req.HTTPCallInformation.QueryValues[client.StatusQueryParamID]) != 1 {
		return fmt.Errorf("missing status: %w", models.ErrInvalidRequest)
	}

	status := req.HTTPCallInformation.QueryValues[client.StatusQueryParamID][0]
	if status != string(client.LinkStatusSuccess) && status != string(client.LinkStatusError) {
		return fmt.Errorf("invalid status: %w", models.ErrInvalidRequest)
	}

	if req.HTTPCallInformation.QueryValues[client.UserIDQueryParamID] == nil ||
		len(req.HTTPCallInformation.QueryValues[client.UserIDQueryParamID]) != 1 {
		return fmt.Errorf("missing user ID: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) completeUserLink(ctx context.Context, req models.CompleteUserLinkRequest) (models.CompleteUserLinkResponse, error) {
	if err := validateCompleteUserLinkRequest(req); err != nil {
		return models.CompleteUserLinkResponse{}, err
	}

	userID := req.HTTPCallInformation.QueryValues[client.UserIDQueryParamID][0]
	status := req.HTTPCallInformation.QueryValues[client.StatusQueryParamID][0]
	switch status {
	case string(client.LinkStatusSuccess):
		connectionID := uuid.New().String()
		err := p.client.CompleteLink(ctx, userID, connectionID)
		if err != nil {
			return models.CompleteUserLinkResponse{}, fmt.Errorf("failed to complete link: %w", err)
		}

		return models.CompleteUserLinkResponse{
			Success: &models.UserLinkSuccessResponse{
				Connections: []models.PSPPsuBankBridgeConnection{
					{
						CreatedAt:    time.Now().UTC(),
						ConnectionID: connectionID,
						AccessToken: &models.Token{
							Token: req.RelatedAttempt.TemporaryToken.Token,
						},
					},
				},
			},
		}, nil

	case string(client.LinkStatusError):
		return models.CompleteUserLinkResponse{
			Error: &models.UserLinkErrorResponse{
				Error: "error",
			},
		}, nil
	}

	// should never happen since we validate the status in the request
	return models.CompleteUserLinkResponse{}, fmt.Errorf("invalid status: %w", models.ErrInvalidRequest)
}
