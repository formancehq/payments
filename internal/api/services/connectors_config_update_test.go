package services

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestConnectorsConfigUpdate(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng, false)
	genericErr := fmt.Errorf("error")

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
			err:           engine.ErrNotFound,
			expectedError: ErrNotFound,
		},
		{
			name:          "other error",
			err:           genericErr,
			expectedError: genericErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := json.RawMessage(`{}`)
			connectorID := models.ConnectorID{}
			eng.EXPECT().UpdateConnector(gomock.Any(), connectorID, config).Return(test.err)
			err := s.ConnectorsConfigUpdate(context.Background(), connectorID, config)
			if test.expectedError == nil {
				require.NoError(t, err)
			} else {
				require.True(t, errors.Is(err, test.expectedError))
			}
		})
	}
}
