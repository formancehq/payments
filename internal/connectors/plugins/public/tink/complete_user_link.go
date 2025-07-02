package tink

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/models"
)

const (
	CredentialIDQueryParamID = "credentials_id"
	StateQueryParamID        = "state"

	ErrorQueryParamID        = "error"
	ErrorMessageQueryParamID = "message"
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

	_, ok = req.HTTPCallInformation.QueryValues[ErrorQueryParamID]
	if ok {
		return nil
	}

	_, ok = req.HTTPCallInformation.QueryValues[CredentialIDQueryParamID]
	if !ok || len(req.HTTPCallInformation.QueryValues[CredentialIDQueryParamID]) != 1 {
		return fmt.Errorf("missing credential IDs: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) completeUserLink(_ context.Context, req models.CompleteUserLinkRequest) (models.CompleteUserLinkResponse, error) {
	if err := validateCompleteUserLinkRequest(req); err != nil {
		return models.CompleteUserLinkResponse{}, err
	}

	errorCode, ok := req.HTTPCallInformation.QueryValues[ErrorQueryParamID]
	if ok {
		// Error callback
		return models.CompleteUserLinkResponse{
			Error: &models.UserLinkErrorResponse{
				Error: fmt.Sprintf("%s: %s", errorCode[0], req.HTTPCallInformation.QueryValues[ErrorMessageQueryParamID][0]),
			},
		}, nil
	}

	return models.CompleteUserLinkResponse{
		Success: &models.UserLinkSuccessResponse{
			Connections: []models.PSUBankBridgeConnection{
				{
					CreatedAt:    time.Now().UTC(),
					ConnectionID: req.HTTPCallInformation.QueryValues[CredentialIDQueryParamID][0],
				},
			},
		},
	}, nil
}
