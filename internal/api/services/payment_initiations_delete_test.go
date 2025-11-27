package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestPaymentInitiationsDelete(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng, false)

	pid := models.PaymentInitiationID{}
	rightLastAdj := models.PaymentInitiationAdjustment{
		Status: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION,
	}
	wrongLastAdj := models.PaymentInitiationAdjustment{
		Status: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
	}
	query := storage.NewListPaymentInitiationAdjustmentsQuery(
		bunpaginate.NewPaginatedQueryOptions(storage.PaymentInitiationAdjustmentsQuery{}).
			WithPageSize(1),
	)

	tests := []struct {
		name                string
		adj                 *models.PaymentInitiationAdjustment
		adjListStorageErr   error
		deleteStorageErr    error
		expectedAdjError    error
		expectedDeleteError error
		typedError          bool
	}{
		{
			name: "success",
			adj:  &rightLastAdj,
		},
		{
			name:             "empty adjustments",
			adj:              nil,
			expectedAdjError: errors.New("payment initiation adjustments not found"),
		},
		{
			name:             "wrong status last adjustment",
			adj:              &wrongLastAdj,
			expectedAdjError: ErrValidation,
			typedError:       true,
		},
		{
			name:              "list adj storage error not found",
			adjListStorageErr: storage.ErrNotFound,
			expectedAdjError:  newStorageError(storage.ErrNotFound, "cannot list payment initiation adjustments"),
		},
		{
			name:              "list adj other error",
			adjListStorageErr: fmt.Errorf("error"),
			expectedAdjError:  newStorageError(fmt.Errorf("error"), "cannot list payment initiation adjustments"),
		},
		{
			name:                "delete pi storage error not found",
			adj:                 &rightLastAdj,
			deleteStorageErr:    storage.ErrNotFound,
			expectedDeleteError: newStorageError(storage.ErrNotFound, "cannot delete payment initiation"),
		},
		{
			name:                "delete pi other error",
			adj:                 &rightLastAdj,
			deleteStorageErr:    fmt.Errorf("error"),
			expectedDeleteError: newStorageError(fmt.Errorf("error"), "cannot delete payment initiation"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var data []models.PaymentInitiationAdjustment
			if test.adj != nil {
				data = []models.PaymentInitiationAdjustment{*test.adj}
			}
			store.EXPECT().PaymentInitiationAdjustmentsList(gomock.Any(), pid, query).Return(
				&bunpaginate.Cursor[models.PaymentInitiationAdjustment]{
					Data: data,
				},
				test.adjListStorageErr,
			)

			if test.expectedAdjError == nil {
				store.EXPECT().PaymentInitiationsDelete(gomock.Any(), pid).Return(test.deleteStorageErr)
			}

			err := s.PaymentInitiationsDelete(context.Background(), pid)
			if test.expectedAdjError == nil && test.expectedDeleteError == nil {
				require.NoError(t, err)
			}
			if test.expectedAdjError != nil {
				if test.typedError {
					require.ErrorIs(t, err, test.expectedAdjError)
				} else {
					require.Equal(t, test.expectedAdjError.Error(), err.Error())
				}
			}
			if test.expectedDeleteError != nil {
				if test.typedError {
					require.ErrorIs(t, err, test.expectedDeleteError)
				} else {
					require.Equal(t, test.expectedDeleteError.Error(), err.Error())
				}
			}
		})
	}
}
