package storage

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/app/storage/models"
)

func (s *Storage) ListPayments(ctx context.Context, sort Sorter, pagination Paginator) ([]*models.Payment, error) {
	var payments []*models.Payment

	query := s.db.NewSelect().Model(&payments)

	if sort != nil {
		query = sort.apply(query)
	}

	query = pagination.apply(query)

	err := query.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}

	return payments, nil
}

func (s *Storage) GetPayment(ctx context.Context, reference string) (*models.Payment, error) {
	var payment *models.Payment

	err := s.db.NewSelect().Model(payment).
		Where("reference = ?", reference).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment %s: %w", reference, err)
	}

	return payment, nil
}
