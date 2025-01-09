package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestPaymentInitiationsReject(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng)

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
		rejectStorageErr    error
		expectedAdjError    error
		expectedRejectError error
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
			rejectStorageErr:    storage.ErrNotFound,
			expectedRejectError: newStorageError(storage.ErrNotFound, "cannot reject payment initiation"),
		},
		{
			name:                "delete pi other error",
			adj:                 &rightLastAdj,
			rejectStorageErr:    fmt.Errorf("error"),
			expectedRejectError: newStorageError(fmt.Errorf("error"), "cannot reject payment initiation"),
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
				store.EXPECT().PaymentInitiationAdjustmentsUpsert(gomock.Any(), gomock.Any()).Return(test.rejectStorageErr)
			}

			err := s.PaymentInitiationsReject(context.Background(), pid)
			if test.expectedAdjError == nil && test.expectedRejectError == nil {
				require.NoError(t, err)
			}
			if test.expectedAdjError != nil {
				if test.typedError {
					require.ErrorIs(t, err, test.expectedAdjError)
				} else {
					require.Equal(t, test.expectedAdjError.Error(), err.Error())
				}
			}
			if test.expectedRejectError != nil {
				if test.typedError {
					require.ErrorIs(t, err, test.expectedRejectError)
				} else {
					require.Equal(t, test.expectedRejectError.Error(), err.Error())
				}
			}
		})
	}
}