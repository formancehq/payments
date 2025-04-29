package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func (s *Service) PaymentServiceUsersGet(ctx context.Context, id uuid.UUID) (*models.PaymentServiceUser, error) {
	psu, err := s.storage.PaymentServiceUsersGet(ctx, id)
	if err != nil {
		return nil, newStorageError(err, "cannot get payment service user")
	}

	return psu, nil
}
