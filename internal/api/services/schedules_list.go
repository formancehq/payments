package services

import (
	"context"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) SchedulesList(ctx context.Context, query storage.ListSchedulesQuery) (*bunpaginate.Cursor[models.Schedule], error) {
	cursor, err := s.storage.SchedulesList(ctx, query)
	return cursor, newStorageError(err, "cannot list schedules")
}
