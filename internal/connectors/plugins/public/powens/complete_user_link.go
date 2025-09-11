package powens

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

const (
	ConnectionIDsQueryParamID = "connection_ids"
	ConnectionIDQueryParamID  = "connection_id"
	StateQueryParamID         = "state"
	ErrorQueryParamID         = "error"
)

func validateCompleteUserLinkRequest(req models.CompleteUserLinkRequest) error {
	if req.RelatedAttempt == nil {
		return fmt.Errorf("related attempt is required: %w", models.ErrInvalidRequest)
	}

	queryState, ok := req.HTTPCallInformation.QueryValues[StateQueryParamID]
	if !ok || len(queryState) != 1 {
		return fmt.Errorf("missing state: %w", models.ErrInvalidRequest)
	}

	decodedState, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(queryState[0])
	if err != nil {
		return fmt.Errorf("failed to decode state: %w", err)
	}

	callbackState := models.CallbackState{}
	if err := json.Unmarshal(decodedState, &callbackState); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	if callbackState.Randomized != req.RelatedAttempt.State.Randomized {
		return fmt.Errorf("state mismatch: %w", models.ErrInvalidRequest)
	}

	if callbackState.AttemptID != req.RelatedAttempt.ID {
		return fmt.Errorf("attempt ID mismatch: %w", models.ErrInvalidRequest)
	}

	connectionIDs, okConnectionIDs := req.HTTPCallInformation.QueryValues[ConnectionIDsQueryParamID]
	errors, okError := req.HTTPCallInformation.QueryValues[ErrorQueryParamID]
	connectionID, okConnectionID := req.HTTPCallInformation.QueryValues[ConnectionIDQueryParamID]
	switch {
	case okError && len(errors) > 0:
	case okConnectionIDs && len(connectionIDs) > 0:
	case okConnectionID && len(connectionID) > 0:
	default:
		return fmt.Errorf("missing connection IDs or error: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) completeUserLink(_ context.Context, req models.CompleteUserLinkRequest) (models.CompleteUserLinkResponse, error) {
	if err := validateCompleteUserLinkRequest(req); err != nil {
		return models.CompleteUserLinkResponse{}, err
	}

	_, okConnectionIDs := req.HTTPCallInformation.QueryValues[ConnectionIDsQueryParamID]
	errors, okError := req.HTTPCallInformation.QueryValues[ErrorQueryParamID]
	_, okConnectionID := req.HTTPCallInformation.QueryValues[ConnectionIDQueryParamID]

	switch {
	case okError:
		return models.CompleteUserLinkResponse{
			Error: &models.UserLinkErrorResponse{
				Error: errors[0],
			},
		}, nil

	case okConnectionIDs, okConnectionID:
		// Here, we don't need to return the connections as they will be created
		// directly by the webhooks.
		// Handling the creation of the connections through the webhooks instead
		// allows us to handle the creation only at one place, and will prevent
		// the need to handle that the user exited the authentication flow just
		// before the redirect.
		return models.CompleteUserLinkResponse{
			Success: &models.UserLinkSuccessResponse{
				Connections: []models.PSPOpenBankingConnection{},
			},
		}, nil

	default:
		// Should not happen since we check the query values in validateCompleteUserLinkRequest
		return models.CompleteUserLinkResponse{}, fmt.Errorf("missing connection IDs or error: %w", models.ErrInvalidRequest)
	}

}
