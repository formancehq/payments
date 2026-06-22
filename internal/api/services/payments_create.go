package services

import (
	"context"

	"github.com/formancehq/payments/pkg/domain/models"
)

func (s *Service) PaymentsCreate(ctx context.Context, payment models.Payment) error {
	return handleEngineErrors(s.engine.CreateFormancePayment(ctx, payment))
}
