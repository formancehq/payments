package services

import (
	"context"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) ConnectorsList(ctx context.Context, query storage.ListConnectorsQuery) (*bunpaginate.Cursor[models.Connector], error) {
	cursor, err := s.storage.ConnectorsList(ctx, query)
	if err != nil {
		return nil, newStorageError(err, "cannot list connectors")
	}
	enrichConnectorsWithType(cursor.Data)
	return cursor, nil
}

func enrichConnectorsWithType(connectors []models.Connector) {
	for i := range connectors {
		if pt, err := registry.GetPluginType(connectors[i].Provider); err == nil {
			connectors[i].ConnectorType = pt
		}
	}
}
