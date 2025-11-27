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

func TestPSUUserLinkAttemptsGet(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng, false)

	id := uuid.New()

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
			expectedError: newStorageError(storage.ErrNotFound, "cannot get payment service users link attempt"),
		},
		{
			name:          "other error",
			err:           fmt.Errorf("error"),
			expectedError: newStorageError(fmt.Errorf("error"), "cannot get payment service users link attempt"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), id).Return(&models.PaymentServiceUser{}, nil)
			store.EXPECT().ConnectorsGet(gomock.Any(), models.ConnectorID{}).Return(&models.Connector{}, nil)
			store.EXPECT().OpenBankingConnectionAttemptsGet(gomock.Any(), id).Return(&models.OpenBankingConnectionAttempt{}, test.err)
			attempt, err := s.PaymentServiceUsersLinkAttemptsGet(context.Background(), id, models.ConnectorID{}, id)
			if test.expectedError == nil {
				require.NotNil(t, attempt)
				require.NoError(t, err)
			} else {
				require.Equal(t, test.expectedError, err)
			}
		})
	}
}
