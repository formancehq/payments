package services

import (
	"context"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) ConversionsList(ctx context.Context, query storage.ListConversionsQuery) (*bunpaginate.Cursor[models.Conversion], error) {
	cursor, err := s.storage.ConversionsList(ctx, query)
	if err != nil {
		return nil, newStorageError(err, "cannot list conversions")
	}

	return cursor, nil
}
