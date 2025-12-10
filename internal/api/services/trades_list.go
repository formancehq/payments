package services

import (
	"context"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) TradesList(ctx context.Context, query storage.ListTradesQuery) (*bunpaginate.Cursor[models.Trade], error) {
	return s.storage.TradesList(ctx, query)
}

