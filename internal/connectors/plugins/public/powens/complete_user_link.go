package powens

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/models"
)

const (
	ConnectionIDsQueryParamID = "connection_ids"
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
	switch {
	case okError && len(errors) > 0:
	case okConnectionIDs && len(connectionIDs) > 0:
	default:
		return fmt.Errorf("missing connection IDs or error: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) completeUserLink(_ context.Context, req models.CompleteUserLinkRequest) (models.CompleteUserLinkResponse, error) {
	if err := validateCompleteUserLinkRequest(req); err != nil {
		return models.CompleteUserLinkResponse{}, err
	}

	connectionIDs, okConnectionIDs := req.HTTPCallInformation.QueryValues[ConnectionIDsQueryParamID]
	errors, okError := req.HTTPCallInformation.QueryValues[ErrorQueryParamID]

	switch {
	case okError:
		return models.CompleteUserLinkResponse{
			Error: &models.UserLinkErrorResponse{
				Error: errors[0],
			},
		}, nil

	case okConnectionIDs:
		connections := make([]models.PSPPsuBankBridgeConnection, len(connectionIDs))
		for i, connectionID := range connectionIDs {
			connections[i] = models.PSPPsuBankBridgeConnection{
				ConnectionID: connectionID,
				CreatedAt:    time.Now().UTC(),
			}
		}

		return models.CompleteUserLinkResponse{
			Success: &models.UserLinkSuccessResponse{
				Connections: connections,
			},
		}, nil

	default:
		// Should not happen since we check the query values in validateCompleteUserLinkRequest
		return models.CompleteUserLinkResponse{}, fmt.Errorf("missing connection IDs or error: %w", models.ErrInvalidRequest)
	}

}
