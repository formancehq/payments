package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func (s *Service) PaymentServiceUsersLinkAttemptsGet(ctx context.Context, id uuid.UUID) (*models.PSUBankBridgeConnectionAttempt, error) {
	attempt, err := s.storage.PSUBankBridgeConnectionAttemptsGet(ctx, id)
	if err != nil {
		return nil, newStorageError(err, "cannot get payment service users link attempt")
	}

	return attempt, nil
}
