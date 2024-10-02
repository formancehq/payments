package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) PaymentsGet(ctx context.Context, id models.PaymentID) (*models.Payment, error) {
	p, err := s.storage.PaymentsGet(ctx, id)
	if err != nil {
		return nil, newStorageError(err, "cannot get payment")
	}

	return p, nil
}
