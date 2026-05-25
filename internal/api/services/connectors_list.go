package services

import (
	"context"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) ConnectorsList(ctx context.Context, query storage.ListConnectorsQuery) (*paginate.Cursor[models.Connector], error) {
	cursor, err := s.storage.ConnectorsList(ctx, query)
	return cursor, newStorageError(err, "cannot list connectors")
}
