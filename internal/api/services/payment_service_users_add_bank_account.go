package services

import (
	"context"

	"github.com/google/uuid"
)

func (s *Service) PaymentServiceUsersAddBankAccount(ctx context.Context, psuID uuid.UUID, bankAccountID uuid.UUID) error {
	return newStorageError(s.storage.PaymentServiceUsersAddBankAccount(ctx, psuID, bankAccountID), "failed to add bank account to payment service user")
}
