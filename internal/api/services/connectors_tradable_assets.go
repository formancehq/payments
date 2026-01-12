package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) ConnectorsGetTradableAssets(ctx context.Context, connectorID models.ConnectorID) ([]models.TradableAsset, error) {
	assets, err := s.engine.GetTradableAssets(ctx, connectorID)
	if err != nil {
		return nil, err
	}
	return assets, nil
}
