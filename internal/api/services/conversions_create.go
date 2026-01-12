package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) ConversionsCreate(ctx context.Context, conversion models.Conversion) error {
	// Store the conversion
	err := s.storage.ConversionsUpsert(ctx, []models.Conversion{conversion})
	if err != nil {
		return newStorageError(err, "cannot create conversion")
	}

	return nil
}
