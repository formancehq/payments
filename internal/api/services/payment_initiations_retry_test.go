package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestPaymentInitiationsRetry(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng, false)

	query := storage.NewListPaymentInitiationAdjustmentsQuery(
		bunpaginate.NewPaginatedQueryOptions(storage.PaymentInitiationAdjustmentsQuery{}).
			WithPageSize(50),
	)
	pid := models.PaymentInitiationID{}
	rightLastAdj := models.PaymentInitiationAdjustment{
		Status: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
	}
	wrongLastAdj := models.PaymentInitiationAdjustment{
		Status: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION,
	}
	piTransfer := models.PaymentInitiation{
		Type: models.PAYMENT_INITIATION_TYPE_TRANSFER,
	}
	piPayout := models.PaymentInitiation{
		Type: models.PAYMENT_INITIATION_TYPE_PAYOUT,
	}

	tests := []struct {
		name                string
		adj                 *models.PaymentInitiationAdjustment
		pi                  models.PaymentInitiation
		engineErr           error
		adjListStorageErr   error
		piGetStorageErr     error
		expectedAdjError    error
		expectedPIError     error
		expectedEngineError error
		typedError          bool
	}{
		{
			name: "success transfer",
			adj:  &rightLastAdj,
			pi:   piTransfer,
		},
		{
			name: "success payout",
			adj:  &rightLastAdj,
			pi:   piPayout,
		},
		{
			name:             "empty adjustments",
			adj:              nil,
			expectedAdjError: errors.New("payment initiation adjustments not found"),
		},
		{
			name:             "wrong status for last adjustment",
			adj:              &wrongLastAdj,
			expectedAdjError: ErrValidation,
			typedError:       true,
		},
		{
			name:                "validation error",
			adj:                 &rightLastAdj,
			pi:                  piPayout,
			engineErr:           engine.ErrValidation,
			expectedEngineError: ErrValidation,
			typedError:          true,
		},
		{
			name:                "not found error",
			adj:                 &rightLastAdj,
			pi:                  piPayout,
			engineErr:           engine.ErrNotFound,
			expectedEngineError: ErrNotFound,
			typedError:          true,
		},
		{
			name:                "other error",
			adj:                 &rightLastAdj,
			pi:                  piPayout,
			engineErr:           fmt.Errorf("error"),
			expectedEngineError: fmt.Errorf("error"),
		},
		{
			name:              "storage error not found",
			adjListStorageErr: storage.ErrNotFound,
			expectedAdjError:  newStorageError(storage.ErrNotFound, "cannot list payment initiation adjustments"),
		},
		{
			name:              "other error",
			adjListStorageErr: fmt.Errorf("error"),
			expectedAdjError:  newStorageError(fmt.Errorf("error"), "cannot list payment initiation adjustments"),
		},
		{
			name:            "storage error not found",
			adj:             &rightLastAdj,
			piGetStorageErr: storage.ErrNotFound,
			expectedPIError: newStorageError(storage.ErrNotFound, "cannot get payment initiation"),
		},
		{
			name:            "other error",
			adj:             &rightLastAdj,
			piGetStorageErr: fmt.Errorf("error"),
			expectedPIError: newStorageError(fmt.Errorf("error"), "cannot get payment initiation"),
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
					PageSize: 1,
					HasMore:  false,
					Data:     data,
				}, test.adjListStorageErr,
			)

			if test.expectedAdjError == nil {
				store.EXPECT().PaymentInitiationsGet(gomock.Any(), pid).Return(&test.pi, test.piGetStorageErr)

				if test.piGetStorageErr == nil {
					switch test.pi.Type {
					case models.PAYMENT_INITIATION_TYPE_TRANSFER:
						eng.EXPECT().CreateTransfer(gomock.Any(), pid, 0*time.Second, 2, false).Return(models.Task{}, test.engineErr)
					case models.PAYMENT_INITIATION_TYPE_PAYOUT:
						eng.EXPECT().CreatePayout(gomock.Any(), pid, gomock.Any(), 2, false).Return(models.Task{}, test.engineErr)
					}
				}
			}

			_, err := s.PaymentInitiationsRetry(context.Background(), pid, false)
			switch {
			case test.expectedAdjError == nil && test.expectedPIError == nil && test.expectedEngineError == nil:
				require.NoError(t, err)
			case test.expectedAdjError != nil:
				if test.typedError {
					require.ErrorIs(t, err, test.expectedAdjError)
				} else {
					require.Equal(t, test.expectedAdjError.Error(), err.Error())
				}
			case test.expectedPIError != nil:
				require.Equal(t, test.expectedPIError.Error(), err.Error())
			case test.expectedEngineError != nil:
				if test.typedError {
					require.ErrorIs(t, err, test.expectedEngineError)
				} else {
					require.Equal(t, test.expectedEngineError, err)
				}
			}
		})
	}
}
