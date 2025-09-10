package services

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) PaymentServiceUsersCompleteLinkFlow(ctx context.Context, connectorID models.ConnectorID, httpCallInformation models.HTTPCallInformation) (string, error) {
	states, ok := httpCallInformation.QueryValues[models.StateQueryParamID]
	if !ok || len(states) != 1 {
		return "", fmt.Errorf("state is missing")
	}

	state, err := models.CallbackStateFromString(states[0])
	if err != nil {
		return "", fmt.Errorf("failed to parse state: %w", err)
	}

	attempt, err := s.storage.OpenBankingConnectionAttemptsGet(ctx, state.AttemptID)
	if err != nil {
		return "", newStorageError(err, "failed to get attempt")
	}

	err = s.engine.CompletePaymentServiceUserLink(ctx, connectorID, state.AttemptID, httpCallInformation)
	if err != nil {
		return "", handleEngineErrors(err)
	}

	return *attempt.ClientRedirectURL, nil
}
