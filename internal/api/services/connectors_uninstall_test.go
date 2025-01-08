package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestConnectorsUninstall(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng)

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
			name:          "other error",
			engineErr:     fmt.Errorf("error"),
			expectedError: fmt.Errorf("error"),
		},
		{
			name:          "storage error not found",
			storageErr:    storage.ErrNotFound,
			expectedError: newStorageError(storage.ErrNotFound, "cannot get connector"),
		},
		{
			name:          "other error",
			storageErr:    fmt.Errorf("error"),
			expectedError: newStorageError(fmt.Errorf("error"), "cannot get connector"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store.EXPECT().ConnectorsGet(gomock.Any(), models.ConnectorID{}).Return(&models.Connector{}, test.storageErr)
			if test.storageErr == nil {
				eng.EXPECT().UninstallConnector(gomock.Any(), models.ConnectorID{}).Return(models.Task{}, test.engineErr)
			}
			_, err := s.ConnectorsUninstall(context.Background(), models.ConnectorID{})
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
