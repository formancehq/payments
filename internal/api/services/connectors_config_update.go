package services

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) ConnectorsConfigUpdate(ctx context.Context, connectorID models.ConnectorID, rawConfig json.RawMessage) error {
	err := s.engine.UpdateConnector(ctx, connectorID, rawConfig)
	if err != nil {
		return handleEngineErrors(err)
	}
	return nil
}
