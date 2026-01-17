package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) ConnectorsGetOrderBook(ctx context.Context, connectorID models.ConnectorID, pair string, depth int) (*models.OrderBook, error) {
	orderBook, err := s.engine.GetOrderBook(ctx, connectorID, pair, depth)
	if err != nil {
		return nil, err
	}
	return orderBook, nil
}
