package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func (s *Service) CounterPartiesGet(ctx context.Context, id uuid.UUID) (*models.CounterParty, error) {
	cp, err := s.storage.CounterPartiesGet(ctx, id)
	if err != nil {
		return nil, newStorageError(err, "cannot get counter party")
	}

	return cp, nil
}
