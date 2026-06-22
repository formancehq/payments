package services

import (
	"context"

	"github.com/formancehq/payments/pkg/domain/models"
)

func (s *Service) PaymentInitiationsGet(ctx context.Context, id models.PaymentInitiationID) (*models.PaymentInitiation, error) {
	pi, err := s.storage.PaymentInitiationsGet(ctx, id)
	if err != nil {
		return nil, newStorageError(err, "cannot get payment initiation")
	}

	return pi, nil
}
