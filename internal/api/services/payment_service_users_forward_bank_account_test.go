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
	gomock "go.uber.org/mock/gomock"
)

func TestPaymentServiceUsersForwardBankAccountsToConnector(t *testing.T) {
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
		name                  string
		bankAccountID         uuid.UUID
		psuID                 uuid.UUID
		engineErr             error
		bankAccountStorageErr error
		psuStorageErr         error
		expectedEngineError   error
		expectedStorageError  error
		typedError            bool
	}{
		{
			name:          "success",
			bankAccountID: uuid.New(),
			psuID:         uuid.New(),
			engineErr:     nil,
		},
		{
			name:                "validation error",
			bankAccountID:       uuid.New(),
			psuID:               uuid.New(),
			engineErr:           engine.ErrValidation,
			expectedEngineError: ErrValidation,
			typedError:          true,
		},
		{
			name:                "not found error",
			bankAccountID:       uuid.New(),
			psuID:               uuid.New(),
			engineErr:           engine.ErrNotFound,
			expectedEngineError: ErrNotFound,
			typedError:          true,
		},
		{
			name:                "other error",
			bankAccountID:       uuid.New(),
			psuID:               uuid.New(),
			engineErr:           fmt.Errorf("error"),
			expectedEngineError: fmt.Errorf("error"),
		},
		{
			name:                  "bank account storage error not found",
			bankAccountStorageErr: storage.ErrNotFound,
			typedError:            true,
			expectedStorageError:  newStorageError(storage.ErrNotFound, "failed to get bank account"),
		},
		{
			name:                  "bank account other error",
			bankAccountStorageErr: fmt.Errorf("error"),
			expectedStorageError:  newStorageError(fmt.Errorf("error"), "failed to get bank account"),
		},
		{
			name:                 "psu storage error not found",
			psuStorageErr:        storage.ErrNotFound,
			typedError:           true,
			expectedStorageError: newStorageError(storage.ErrNotFound, "failed to get payment service user"),
		},
		{
			name:                 "psu other error",
			psuStorageErr:        fmt.Errorf("error"),
			expectedStorageError: newStorageError(fmt.Errorf("error"), "failed to get payment service user"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store.EXPECT().BankAccountsGet(gomock.Any(), test.bankAccountID, true).Return(&models.BankAccount{}, test.bankAccountStorageErr)

			if test.bankAccountStorageErr == nil {
				store.EXPECT().PaymentServiceUsersGet(gomock.Any(), test.psuID).Return(&models.PaymentServiceUser{}, test.psuStorageErr)

				if test.psuStorageErr == nil {
					eng.EXPECT().ForwardBankAccount(gomock.Any(), models.BankAccount{}, connectorID, false).Return(models.Task{}, test.engineErr)
				}
			}
			_, err := s.PaymentServiceUsersForwardBankAccountToConnector(context.Background(), test.psuID, test.bankAccountID, connectorID)
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
