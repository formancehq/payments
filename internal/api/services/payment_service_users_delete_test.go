package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	gomock "github.com/golang/mock/gomock"
)

func TestPSUDelete(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng, false)

	id := uuid.New()

	tests := []struct {
		name          string
		engineErr     error
		storageErr    error
		expectedError error
		typedError    bool
	}{
		{
			name:          "success",
			engineErr:     nil,
			storageErr:    nil,
			expectedError: nil,
		},
		{
			name:          "validation error",
			engineErr:     engine.ErrValidation,
			expectedError: ErrValidation,
			typedError:    true,
		},
		{
			name:          "not found error",
			engineErr:     engine.ErrNotFound,
			expectedError: ErrNotFound,
			typedError:    true,
		},
		{
			name:          "engine other error",
			engineErr:     fmt.Errorf("error"),
			expectedError: fmt.Errorf("error"),
		},
		{
			name:          "storage not found error",
			storageErr:    storage.ErrNotFound,
			expectedError: newStorageError(storage.ErrNotFound, "cannot get payment service user"),
		},
		{
			name:          "other storage error",
			storageErr:    fmt.Errorf("error"),
			expectedError: newStorageError(fmt.Errorf("error"), "cannot get payment service user"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), id).Return(&models.PaymentServiceUser{}, test.storageErr)
			if test.storageErr == nil {
				eng.EXPECT().DeletePaymentServiceUser(gomock.Any(), id).Return(models.Task{}, test.engineErr)
			}
			_, err := s.PaymentServiceUsersDelete(context.Background(), id)
			if test.expectedError == nil {
				require.NoError(t, err)
			} else if test.typedError {
				require.ErrorIs(t, err, test.expectedError)
			} else {
				require.Equal(t, test.expectedError, err)
			}
		})
	}
}
