package storage

import (
	"context"

	"github.com/formancehq/payments/internal/app/storage/models"
)

func (s *Storage) ListConnectors(ctx context.Context) ([]*models.Connector, error) {
	var res []*models.Connector
	err := s.db.NewSelect().Model(&res).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return res, nil
}
