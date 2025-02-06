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

func TestTasksGet(t *testing.T) {
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
			expectedError: newStorageError(storage.ErrNotFound, "cannot get task"),
		},
		{
			name:          "other error",
			err:           fmt.Errorf("error"),
			expectedError: newStorageError(fmt.Errorf("error"), "cannot get task"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store.EXPECT().TasksGet(gomock.Any(), models.TaskID{}).Return(&models.Task{}, test.err)
			payment, err := s.TaskGet(context.Background(), models.TaskID{})
			if test.expectedError == nil {
				require.NotNil(t, payment)
				require.NoError(t, err)
			} else {
				require.Equal(t, test.expectedError, err)
			}
		})
	}
}
