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

func TestPSUConnectionsList(t *testing.T) {
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
			expectedError: newStorageError(storage.ErrNotFound, "cannot list psu bank bridge connections"),
		},
		{
			name:          "other error",
			err:           fmt.Errorf("error"),
			expectedError: newStorageError(fmt.Errorf("error"), "cannot list psu bank bridge connections"),
		},
	}

	id := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "plaid",
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := storage.ListPsuBankBridgeConnectionsQuery{}
			store.EXPECT().PSUBankBridgeConnectionsList(gomock.Any(), id, &connectorID, query).Return(nil, test.err)
			_, err := s.PaymentServiceUsersConnectionsList(context.Background(), id, &connectorID, query)
			if test.expectedError == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, test.expectedError, err)
			}
		})
	}
}
