package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) TradesGet(ctx context.Context, id models.TradeID) (*models.Trade, error) {
	return s.storage.TradesGet(ctx, id)
}

