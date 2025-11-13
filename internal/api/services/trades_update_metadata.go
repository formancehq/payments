package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) TradesUpdateMetadata(ctx context.Context, id models.TradeID, metadata map[string]string) error {
	return s.storage.TradesUpdateMetadata(ctx, id, metadata)
}

