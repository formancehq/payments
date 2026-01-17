package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) ConnectorsGetOHLC(ctx context.Context, connectorID models.ConnectorID, req models.GetOHLCRequest) (*models.OHLCData, error) {
	ohlc, err := s.engine.GetOHLC(ctx, connectorID, req)
	if err != nil {
		return nil, err
	}
	return ohlc, nil
}
