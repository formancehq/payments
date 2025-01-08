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

func TestPaymentInitiationAdjustmentsGetLast(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng)

	pid := models.PaymentInitiationID{}

	tests := []struct {
		name            string
		storageResponse []models.PaymentInitiationAdjustment
		err             error
		expectedError   error
	}{
		{
			name:            "success",
			err:             nil,
			storageResponse: []models.PaymentInitiationAdjustment{{}},
			expectedError:   nil,
		},
		{
			name:            "no adjustments",
			err:             nil,
			storageResponse: []models.PaymentInitiationAdjustment{},
			expectedError:   errors.New("payment initiation's adjustments not found"),
		},
		{
			name:          "storage error not found",
			err:           storage.ErrNotFound,
			expectedError: newStorageError(storage.ErrNotFound, "cannot list payment initiation's adjustments"),
		},
		{
			name:          "other error",
			err:           fmt.Errorf("error"),
			expectedError: newStorageError(fmt.Errorf("error"), "cannot list payment initiation's adjustments"),
		},
	}

	for _, test := range tests {
		query := storage.NewListPaymentInitiationAdjustmentsQuery(
			bunpaginate.NewPaginatedQueryOptions(storage.PaymentInitiationAdjustmentsQuery{}).
				WithPageSize(1),
		)
		store.EXPECT().PaymentInitiationAdjustmentsList(gomock.Any(), pid, query).Return(&bunpaginate.Cursor[models.PaymentInitiationAdjustment]{
			PageSize: 1,
			HasMore:  false,
			Data:     test.storageResponse,
		}, test.err)
		adj, err := s.PaymentInitiationAdjustmentsGetLast(context.Background(), pid)
		if test.expectedError == nil {
			require.NotNil(t, adj)
			require.NoError(t, err)
		} else {
			require.Equal(t, test.expectedError.Error(), err.Error())
		}
	}
}
