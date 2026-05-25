package services

import (
	"context"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) WorkflowsInstancesList(ctx context.Context, query storage.ListInstancesQuery) (*paginate.Cursor[models.Instance], error) {
	cursor, err := s.storage.InstancesList(ctx, query)
	return cursor, newStorageError(err, "cannot list instances")
}
