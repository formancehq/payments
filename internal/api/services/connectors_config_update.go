package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) ConnectorsConfigUpdate(ctx context.Context, connector models.Connector) error {
	err := s.storage.ConnectorsConfigUpdate(ctx, connector)
	if err != nil {
		return newStorageError(err, "update connector")
	}
	return nil
}
