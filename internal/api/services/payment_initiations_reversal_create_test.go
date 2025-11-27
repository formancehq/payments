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

func TestPaymentInitiationsReversalCreate(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng, false)

	pid := models.PaymentInitiationID{}
	piTransfer := models.PaymentInitiation{
		Type:  models.PAYMENT_INITIATION_TYPE_TRANSFER,
		Asset: "USD/2",
	}
	piPayout := models.PaymentInitiation{
		Type:  models.PAYMENT_INITIATION_TYPE_PAYOUT,
		Asset: "USD/2",
	}

	tests := []struct {
		name                string
		asset               string
		pi                  models.PaymentInitiation
		engineErr           error
		piGetStorageErr     error
		piUpsertStorageErr  error
		expectedPIError     error
		expectedUpsertError error
		expectedEngineError error
		typedError          bool
	}{
		{
			name:  "success transfer",
			asset: "USD/2",
			pi:    piTransfer,
		},
		{
			name:  "success payout",
			asset: "USD/2",
			pi:    piPayout,
		},
		{
			name:            "wrong asset",
			asset:           "EUR/2",
			pi:              piTransfer,
			expectedPIError: ErrValidation,
			typedError:      true,
		},
		{
			name:                "validation error",
			asset:               "USD/2",
			pi:                  piPayout,
			engineErr:           engine.ErrValidation,
			expectedEngineError: ErrValidation,
			typedError:          true,
		},
		{
			name:                "not found error",
			asset:               "USD/2",
			pi:                  piPayout,
			engineErr:           engine.ErrNotFound,
			expectedEngineError: ErrNotFound,
			typedError:          true,
		},
		{
			name:                "other error",
			asset:               "USD/2",
			pi:                  piPayout,
			engineErr:           fmt.Errorf("error"),
			expectedEngineError: fmt.Errorf("error"),
		},
		{
			name:            "get storage error not found",
			asset:           "USD/2",
			piGetStorageErr: storage.ErrNotFound,
			expectedPIError: newStorageError(storage.ErrNotFound, "cannot get payment initiation"),
		},
		{
			name:            "get other error",
			asset:           "USD/2",
			piGetStorageErr: fmt.Errorf("error"),
			expectedPIError: newStorageError(fmt.Errorf("error"), "cannot get payment initiation"),
		},
		{
			name:                "upsert storage error not found",
			pi:                  piPayout,
			asset:               "USD/2",
			piUpsertStorageErr:  storage.ErrNotFound,
			expectedUpsertError: newStorageError(storage.ErrNotFound, "cannot create payment initiation reversal"),
		},
		{
			name:                "upsert other error",
			asset:               "USD/2",
			pi:                  piPayout,
			piUpsertStorageErr:  fmt.Errorf("error"),
			expectedUpsertError: newStorageError(fmt.Errorf("error"), "cannot create payment initiation reversal"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store.EXPECT().PaymentInitiationsGet(gomock.Any(), pid).Return(&test.pi, test.piGetStorageErr)
			if test.expectedPIError == nil {
				store.EXPECT().PaymentInitiationReversalsUpsert(gomock.Any(), gomock.Any(), gomock.Any()).Return(test.piUpsertStorageErr)

				if test.expectedUpsertError == nil {
					switch test.pi.Type {
					case models.PAYMENT_INITIATION_TYPE_TRANSFER:
						eng.EXPECT().ReverseTransfer(gomock.Any(), gomock.Any(), false).Return(models.Task{}, test.engineErr)
					case models.PAYMENT_INITIATION_TYPE_PAYOUT:
						eng.EXPECT().ReversePayout(gomock.Any(), gomock.Any(), false).Return(models.Task{}, test.engineErr)
					}
				}
			}

			_, err := s.PaymentInitiationReversalsCreate(context.Background(), models.PaymentInitiationReversal{Asset: test.asset}, false)
			switch {
			case test.expectedPIError == nil && test.expectedUpsertError == nil && test.expectedEngineError == nil:
				require.NoError(t, err)
			case test.expectedUpsertError != nil:
				if test.typedError {
					require.ErrorIs(t, err, test.expectedUpsertError)
				} else {
					require.Equal(t, test.expectedUpsertError.Error(), err.Error())
				}
			case test.expectedPIError != nil:
				if test.typedError {
					require.ErrorIs(t, err, test.expectedPIError)
				} else {
					require.Equal(t, test.expectedPIError.Error(), err.Error())
				}
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
