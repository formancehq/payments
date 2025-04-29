package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) PaymentServiceUsersCreate(ctx context.Context, psu models.PaymentServiceUser) error {
	return newStorageError(s.storage.PaymentServiceUsersCreate(ctx, psu), "cannot create payment service user")
}
