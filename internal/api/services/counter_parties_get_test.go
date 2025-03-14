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

func TestCounterPartiesGet(t *testing.T) {
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
			expectedError: newStorageError(storage.ErrNotFound, "cannot get counter party"),
		},
		{
			name:          "other error",
			err:           fmt.Errorf("error"),
			expectedError: newStorageError(fmt.Errorf("error"), "cannot get counter party"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store.EXPECT().CounterPartiesGet(gomock.Any(), id).Return(&models.CounterParty{}, test.err)
			counterParty, err := s.CounterPartiesGet(context.Background(), id)
			if test.expectedError == nil {
				require.NotNil(t, counterParty)
				require.NoError(t, err)
			} else {
				require.Equal(t, test.expectedError, err)
			}
		})
	}
}
