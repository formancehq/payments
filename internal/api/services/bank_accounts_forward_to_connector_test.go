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
		name          string
		bankAccountID uuid.UUID
		err           error
		expectedError error
		typedError    bool
	}{
		{
			name:          "success",
			err:           nil,
			expectedError: nil,
		},
		{
			name:          "validation error",
			err:           engine.ErrValidation,
			expectedError: ErrValidation,
			typedError:    true,
		},
		{
			name:          "not found error",
			err:           engine.ErrNotFound,
			expectedError: ErrNotFound,
			typedError:    true,
		},
		{
			name:          "other error",
			err:           fmt.Errorf("error"),
			expectedError: fmt.Errorf("error"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			eng.EXPECT().ForwardBankAccount(gomock.Any(), test.bankAccountID, connectorID, false).Return(models.Task{}, test.err)
			_, err := s.BankAccountsForwardToConnector(context.Background(), test.bankAccountID, connectorID, false)
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
