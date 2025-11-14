package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	gomock "github.com/golang/mock/gomock"
)

func TestPaymentServiceUsersAddBankAccount(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng, false)

	tests := []struct {
		name          string
		err           error
		expectedError error
	}{
		{
			name:          "success",
			err:           nil,
			expectedError: nil,
		},
		{
			name:          "storage error not found",
			err:           storage.ErrNotFound,
			expectedError: newStorageError(storage.ErrNotFound, "failed to add bank account to payment service user"),
		},
		{
			name:          "other error",
			err:           fmt.Errorf("error"),
			expectedError: newStorageError(fmt.Errorf("error"), "failed to add bank account to payment service user"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			psuID, baID := uuid.New(), uuid.New()
			store.EXPECT().PaymentServiceUsersAddBankAccount(gomock.Any(), psuID, baID).Return(test.err)
			err := s.PaymentServiceUsersAddBankAccount(context.Background(), psuID, baID)
			if test.expectedError == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, test.expectedError, err)
			}
		})
	}
}
