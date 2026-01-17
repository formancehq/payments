package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) ConnectorsGetQuote(ctx context.Context, connectorID models.ConnectorID, req models.GetQuoteRequest) (*models.Quote, error) {
	quote, err := s.engine.GetQuote(ctx, connectorID, req)
	if err != nil {
		return nil, err
	}
	return quote, nil
}
