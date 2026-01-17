package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) ConnectorsGetTicker(ctx context.Context, connectorID models.ConnectorID, pair string) (*models.Ticker, error) {
	ticker, err := s.engine.GetTicker(ctx, connectorID, pair)
	if err != nil {
		return nil, err
	}
	return ticker, nil
}
