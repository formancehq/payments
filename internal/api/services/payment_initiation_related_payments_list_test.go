package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestPaymentInitiationRelatedPaymentsList(t *testing.T) {
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
			expectedError: newStorageError(storage.ErrNotFound, "cannot list payment initiation related payments"),
		},
		{
			name:          "other error",
			err:           fmt.Errorf("error"),
			expectedError: newStorageError(fmt.Errorf("error"), "cannot list payment initiation related payments"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := storage.ListPaymentInitiationRelatedPaymentsQuery{}
			store.EXPECT().PaymentInitiationRelatedPaymentsList(gomock.Any(), pid, query).Return(nil, test.err)
			_, err := s.PaymentInitiationRelatedPaymentsList(context.Background(), pid, query)
			if test.expectedError == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, test.expectedError, err)
			}
		})
	}
}

func TestPaymentInitiationRelatedPaymentsListAll(t *testing.T) {
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
			expectedError: newStorageError(storage.ErrNotFound, "cannot list payment initiation related payments"),
		},
		{
			name:          "other error",
			err:           fmt.Errorf("error"),
			expectedError: newStorageError(fmt.Errorf("error"), "cannot list payment initiation related payments"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := storage.NewListPaymentInitiationRelatedPaymentsQuery(
				bunpaginate.NewPaginatedQueryOptions(storage.PaymentInitiationRelatedPaymentsQuery{}).
					WithPageSize(50),
			)

			store.EXPECT().PaymentInitiationRelatedPaymentsList(gomock.Any(), pid, query).Return(&bunpaginate.Cursor[models.Payment]{
				PageSize: 1,
				HasMore:  false,
				Data:     []models.Payment{{}},
			}, test.err)
			_, err := s.PaymentInitiationRelatedPaymentsListAll(context.Background(), pid)
			if test.expectedError == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, test.expectedError, err)
			}
		})
	}
}
