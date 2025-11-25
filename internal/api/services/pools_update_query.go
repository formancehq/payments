package services

import (
	"context"

	"github.com/google/uuid"
)

func (s *Service) PoolsUpdateQuery(ctx context.Context, id uuid.UUID, query map[string]any) error {
	err := s.engine.UpdatePoolQuery(ctx, id, query)
	return handleEngineErrors(err)
}
