package tink

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/pkg/connector"
)

const (
	CredentialIDQueryParamID = "credentials_id"
	StateQueryParamID        = "state"

	ErrorQueryParamID        = "error"
	ErrorMessageQueryParamID = "message"
)

func validateCompleteUserLinkRequest(req connector.CompleteUserLinkRequest) error {
	if req.RelatedAttempt == nil {
		return fmt.Errorf("related attempt is required: %w", connector.ErrInvalidRequest)
	}

	queryState, ok := req.HTTPCallInformation.QueryValues[StateQueryParamID]
	if !ok || len(queryState) != 1 {
		return fmt.Errorf("missing state: %w", connector.ErrInvalidRequest)
	}

	decodedState, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(queryState[0])
	if err != nil {
		return fmt.Errorf("failed to decode state: %w", err)
	}

	callbackState := connector.CallbackState{}
	if err := json.Unmarshal(decodedState, &callbackState); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	if callbackState.Randomized != req.RelatedAttempt.State.Randomized {
		return fmt.Errorf("state mismatch: %w", connector.ErrInvalidRequest)
	}

	_, ok = req.HTTPCallInformation.QueryValues[ErrorQueryParamID]
	if ok {
		return nil
	}

	_, ok = req.HTTPCallInformation.QueryValues[CredentialIDQueryParamID]
	if !ok || len(req.HTTPCallInformation.QueryValues[CredentialIDQueryParamID]) != 1 ||
		req.HTTPCallInformation.QueryValues[CredentialIDQueryParamID][0] == "" {
		return fmt.Errorf("missing credential IDs: %w", connector.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) completeUserLink(_ context.Context, req connector.CompleteUserLinkRequest) (connector.CompleteUserLinkResponse, error) {
	if err := validateCompleteUserLinkRequest(req); err != nil {
		return connector.CompleteUserLinkResponse{}, err
	}

	errorCode, ok := req.HTTPCallInformation.QueryValues[ErrorQueryParamID]
	if ok {
		errMessage := "got an error from tink"
		if len(errorCode) > 0 {
			errMessage += fmt.Sprintf(": %s", errorCode[0])
		}
		if len(req.HTTPCallInformation.QueryValues[ErrorMessageQueryParamID]) > 0 {
			errMessage += fmt.Sprintf(": %s", req.HTTPCallInformation.QueryValues[ErrorMessageQueryParamID][0])
		}

		// Error callback
		return connector.CompleteUserLinkResponse{
			Error: &connector.UserLinkErrorResponse{
				Error: errMessage,
			},
		}, nil
	}

	return connector.CompleteUserLinkResponse{
		Success: &connector.UserLinkSuccessResponse{
			Connections: []connector.PSPOpenBankingConnection{
				{
					CreatedAt:    time.Now().UTC(),
					ConnectionID: req.HTTPCallInformation.QueryValues[CredentialIDQueryParamID][0],
				},
			},
		},
	}, nil
}
