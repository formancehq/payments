package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) TradesCreate(ctx context.Context, trade models.Trade) error {
	return handleEngineErrors(s.engine.CreateFormanceTrade(ctx, trade))
}

