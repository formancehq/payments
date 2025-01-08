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

func TestPaymentsUpdateMetadata(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng)

	id := models.PaymentID{}

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
			expectedError: newStorageError(storage.ErrNotFound, "cannot update payment metadata"),
		},
		{
			name:          "other error",
			err:           fmt.Errorf("error"),
			expectedError: newStorageError(fmt.Errorf("error"), "cannot update payment metadata"),
		},
	}

	for _, test := range tests {
		store.EXPECT().PaymentsUpdateMetadata(gomock.Any(), id, map[string]string{}).Return(test.err)
		err := s.PaymentsUpdateMetadata(context.Background(), id, map[string]string{})
		if test.expectedError == nil {
			require.NoError(t, err)
		} else {
			require.Equal(t, test.expectedError, err)
		}
	}
}
