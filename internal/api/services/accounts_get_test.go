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

func TestAccountsGet(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng)

	id := models.AccountID{
		Reference: "test",
		ConnectorID: models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "test",
		},
	}

	tests := []struct {
		name          string
		accountID     models.AccountID
		err           error
		expectedError error
	}{
		{
			name:          "success",
			accountID:     id,
			err:           nil,
			expectedError: nil,
		},
		{
			name:          "storage error not found",
			accountID:     id,
			err:           storage.ErrNotFound,
			expectedError: newStorageError(storage.ErrNotFound, "cannot get account"),
		},
		{
			name:          "other error",
			err:           fmt.Errorf("error"),
			expectedError: newStorageError(fmt.Errorf("error"), "cannot get account"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store.EXPECT().AccountsGet(gomock.Any(), test.accountID).Return(&models.Account{}, test.err)
			account, err := s.AccountsGet(context.Background(), test.accountID)
			if test.expectedError == nil {
				require.NotNil(t, account)
				require.NoError(t, err)
			} else {
				require.Equal(t, test.expectedError, err)
			}
		})
	}
}
