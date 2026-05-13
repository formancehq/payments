package teller

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/models"
)

const (
	AccessTokenQueryParam  = "access_token"
	EnrollmentIDQueryParam = "enrollment_id"
)

func validateCompleteUserLinkRequest(req models.CompleteUserLinkRequest) error {
	if req.HTTPCallInformation.QueryValues == nil {
		return fmt.Errorf("missing query values: %w", models.ErrInvalidRequest)
	}

	if _, ok := req.HTTPCallInformation.QueryValues[AccessTokenQueryParam]; !ok {
		return fmt.Errorf("missing access_token: %w", models.ErrInvalidRequest)
	}

	if _, ok := req.HTTPCallInformation.QueryValues[EnrollmentIDQueryParam]; !ok {
		return fmt.Errorf("missing enrollment_id: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) completeUserLink(ctx context.Context, req models.CompleteUserLinkRequest) (models.CompleteUserLinkResponse, error) {
	if err := validateCompleteUserLinkRequest(req); err != nil {
		return models.CompleteUserLinkResponse{}, err
	}

	accessToken := req.HTTPCallInformation.QueryValues[AccessTokenQueryParam][0]
	enrollmentID := req.HTTPCallInformation.QueryValues[EnrollmentIDQueryParam][0]

	return models.CompleteUserLinkResponse{
		Success: &models.UserLinkSuccessResponse{
			Connections: []models.PSPOpenBankingConnection{
				{
					CreatedAt:    time.Now().UTC(),
					ConnectionID: enrollmentID,
					AccessToken: &models.Token{
						Token: accessToken,
					},
				},
			},
		},
	}, nil
}
