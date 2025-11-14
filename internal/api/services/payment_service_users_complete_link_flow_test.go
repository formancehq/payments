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
	gomock "github.com/golang/mock/gomock"
)

func TestPSUCompleteLinkFlow(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng, false)

	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	state := models.CallbackState{
		AttemptID: uuid.New(),
	}

	stateString := state.String()

	httpCallInformation := models.HTTPCallInformation{
		QueryValues: map[string][]string{
			"state": {stateString},
		},
	}

	tests := []struct {
		name                 string
		otherError           error
		engineErr            error
		storageErr           error
		expectedEngineError  error
		expectedStorageError error
		typedError           bool
		httpCallInformation  models.HTTPCallInformation
	}{
		{
			name:                "success",
			engineErr:           nil,
			httpCallInformation: httpCallInformation,
		},
		{
			name:       "missing state",
			otherError: fmt.Errorf("state is missing"),
			typedError: false,
			httpCallInformation: models.HTTPCallInformation{
				QueryValues: map[string][]string{
					"state": {},
				},
			},
		},
		{
			name:                "validation error",
			engineErr:           engine.ErrValidation,
			expectedEngineError: ErrValidation,
			typedError:          true,
			httpCallInformation: httpCallInformation,
		},
		{
			name:                "not found error",
			engineErr:           engine.ErrNotFound,
			expectedEngineError: ErrNotFound,
			typedError:          true,
			httpCallInformation: httpCallInformation,
		},
		{
			name:                "other error",
			engineErr:           fmt.Errorf("error"),
			expectedEngineError: fmt.Errorf("error"),
			httpCallInformation: httpCallInformation,
		},
		{
			name:                 "storage error not found",
			storageErr:           storage.ErrNotFound,
			typedError:           true,
			expectedStorageError: newStorageError(storage.ErrNotFound, "failed to get attempt"),
			httpCallInformation:  httpCallInformation,
		},
		{
			name:                 "other error",
			storageErr:           fmt.Errorf("error"),
			expectedStorageError: newStorageError(fmt.Errorf("error"), "failed to get attempt"),
			httpCallInformation:  httpCallInformation,
		},
	}

	clientRedirectURL := "https://example.com"

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			if test.otherError == nil {
				store.EXPECT().OpenBankingConnectionAttemptsGet(gomock.Any(), gomock.Any()).Return(&models.OpenBankingConnectionAttempt{
					ClientRedirectURL: &clientRedirectURL,
				}, test.storageErr)
			}

			if test.storageErr == nil && test.otherError == nil {
				eng.EXPECT().CompletePaymentServiceUserLink(gomock.Any(), connectorID, gomock.Any(), httpCallInformation).Return(test.engineErr)
			}
			_, err := s.PaymentServiceUsersCompleteLinkFlow(context.Background(), connectorID, test.httpCallInformation)
			switch {
			case test.expectedEngineError != nil && test.typedError:
				require.ErrorIs(t, err, test.expectedEngineError)
			case test.expectedEngineError != nil && !test.typedError:
				require.Error(t, err)
				require.Equal(t, test.expectedEngineError.Error(), err.Error())
			case test.expectedStorageError != nil && test.typedError:
				require.ErrorIs(t, err, test.expectedStorageError)
			case test.expectedStorageError != nil && !test.typedError:
				require.Error(t, err)
				require.Equal(t, test.expectedStorageError.Error(), err.Error())
			case test.otherError != nil:
				require.Error(t, err)
				require.Equal(t, test.otherError.Error(), err.Error())
			default:
				require.NoError(t, err)
			}
		})
	}
}
