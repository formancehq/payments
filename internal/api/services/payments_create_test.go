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

func TestPaymentsCreate(t *testing.T) {
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
			name:          "capability not supported",
			err:           &engine.ErrConnectorCapabilityNotSupported{},
			expectedError: &engine.ErrConnectorCapabilityNotSupported{},
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
			eng.EXPECT().CreateFormancePayment(gomock.Any(), gomock.Any()).Return(test.err)
			err := s.PaymentsCreate(context.Background(), models.Payment{})
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
