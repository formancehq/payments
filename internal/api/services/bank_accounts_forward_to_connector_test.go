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

func TestBankAccountsForwardToConnector(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng, false)

	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	tests := []struct {
		name                 string
		bankAccountID        uuid.UUID
		engineErr            error
		storageErr           error
		expectedEngineError  error
		expectedStorageError error
		typedError           bool
	}{
		{
			name:      "success",
			engineErr: nil,
		},
		{
			name:                "validation error",
			engineErr:           engine.ErrValidation,
			expectedEngineError: ErrValidation,
			typedError:          true,
		},
		{
			name:                "not found error",
			engineErr:           engine.ErrNotFound,
			expectedEngineError: ErrNotFound,
			typedError:          true,
		},
		{
			name:                "other error",
			engineErr:           fmt.Errorf("error"),
			expectedEngineError: fmt.Errorf("error"),
		},
		{
			name:                 "storage error not found",
			storageErr:           storage.ErrNotFound,
			typedError:           true,
			expectedStorageError: newStorageError(storage.ErrNotFound, "failed to get bank account"),
		},
		{
			name:                 "other error",
			storageErr:           fmt.Errorf("error"),
			expectedStorageError: newStorageError(fmt.Errorf("error"), "failed to get bank account"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store.EXPECT().BankAccountsGet(gomock.Any(), test.bankAccountID, true).Return(&models.BankAccount{}, test.storageErr)

			if test.storageErr == nil {
				eng.EXPECT().ForwardBankAccount(gomock.Any(), models.BankAccount{}, connectorID, false).Return(models.Task{}, test.engineErr)
			}
			_, err := s.BankAccountsForwardToConnector(context.Background(), test.bankAccountID, connectorID, false)
			switch {
			case test.expectedEngineError != nil && test.typedError:
				require.ErrorIs(t, err, test.expectedEngineError)
			case test.expectedEngineError != nil && !test.typedError:
				require.Error(t, err)
				require.Equal(t, test.expectedEngineError.Error(), err.Error())
			case test.expectedStorageError != nil && test.typedError:
				require.ErrorIs(t, err, test.expectedStorageError)
			case test.expectedStorageError != nil && !test.typedError:
				require.Error(t, err)
				require.Equal(t, test.expectedStorageError.Error(), err.Error())
			default:
				require.NoError(t, err)
			}
		})
	}
}
