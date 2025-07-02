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

	_, ok = req.HTTPCallInformation.QueryValues[ConnectionIDsQueryParamID]
	if !ok {
		return fmt.Errorf("missing connection IDs: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) completeUserLink(_ context.Context, req models.CompleteUserLinkRequest) (models.CompleteUserLinkResponse, error) {
	if err := validateCompleteUserLinkRequest(req); err != nil {
		return models.CompleteUserLinkResponse{}, err
	}

	connectionIDs := req.HTTPCallInformation.QueryValues[ConnectionIDsQueryParamID]
	connections := make([]models.PSUBankBridgeConnection, len(connectionIDs))
	for i, connectionID := range connectionIDs {
		connections[i] = models.PSUBankBridgeConnection{
			ConnectionID: connectionID,
			CreatedAt:    time.Now().UTC(),
		}
	}

	return models.CompleteUserLinkResponse{
		Success: &models.UserLinkSuccessResponse{
			Connections: connections,
		},
	}, nil
}
