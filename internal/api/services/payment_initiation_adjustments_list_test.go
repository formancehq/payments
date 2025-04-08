package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestPaymentInitiationAdjustmentsList(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng, false)

	pid := models.PaymentInitiationID{}

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
			expectedError: newStorageError(storage.ErrNotFound, "cannot list payment initiation adjustments"),
		},
		{
			name:          "other error",
			err:           fmt.Errorf("error"),
			expectedError: newStorageError(fmt.Errorf("error"), "cannot list payment initiation adjustments"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := storage.ListPaymentInitiationAdjustmentsQuery{}
			store.EXPECT().PaymentInitiationAdjustmentsList(gomock.Any(), pid, query).Return(nil, test.err)
			_, err := s.PaymentInitiationAdjustmentsList(context.Background(), pid, query)
			if test.expectedError == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, test.expectedError, err)
			}
		})
	}
}

func TestPaymentInitiationAdjustmentsListAll(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng, false)

	pid := models.PaymentInitiationID{}

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
			expectedError: newStorageError(storage.ErrNotFound, "cannot list payment initiation adjustments"),
		},
		{
			name:          "other error",
			err:           fmt.Errorf("error"),
			expectedError: newStorageError(fmt.Errorf("error"), "cannot list payment initiation adjustments"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := storage.NewListPaymentInitiationAdjustmentsQuery(
				bunpaginate.NewPaginatedQueryOptions(storage.PaymentInitiationAdjustmentsQuery{}).
					WithPageSize(50),
			)

			store.EXPECT().PaymentInitiationAdjustmentsList(gomock.Any(), pid, query).Return(&bunpaginate.Cursor[models.PaymentInitiationAdjustment]{
				PageSize: 1,
				HasMore:  false,
				Data:     []models.PaymentInitiationAdjustment{{}},
			}, test.err)
			_, err := s.PaymentInitiationAdjustmentsListAll(context.Background(), pid)
			if test.expectedError == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, test.expectedError, err)
			}
		})
	}
}
