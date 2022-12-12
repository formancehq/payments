package storage

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/app/models"
)

func (s *Storage) GetConfig(ctx context.Context, connectorProvider models.ConnectorProvider, destination any) error {
	err := s.db.NewSelect().Model(&models.Connector{}).
		Column("config").
		Where("provider = ?", connectorProvider).
		Scan(ctx, destination)
	if err != nil {
		return fmt.Errorf("failed to get config for connector %s: %w", connectorProvider, err)
	}

	return nil
}
