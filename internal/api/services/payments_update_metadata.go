package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) PaymentsUpdateMetadata(ctx context.Context, id models.PaymentID, metadata map[string]string) error {
	return newStorageError(s.storage.PaymentsUpdateMetadata(ctx, id, metadata), "cannot update payment metadata")
}
