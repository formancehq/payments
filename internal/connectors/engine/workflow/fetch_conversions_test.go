package workflow

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) newPSPConversion() models.PSPConversion {
	return models.PSPConversion{
		Reference:    "test-conversion-" + s.connectorID.Reference.String()[:8],
		CreatedAt:    time.Now().UTC(),
		SourceAsset:  "USD/2",
		TargetAsset:  "BTC/8",
		SourceAmount: big.NewInt(100000),
		TargetAmount: big.NewInt(100000000),
		Status:       models.CONVERSION_STATUS_COMPLETED,
		WalletID:     "test-wallet-id",
		Metadata:     map[string]string{"key": "value"},
		Raw:          []byte(`{}`),
	}
}

func (s *UnitTestSuite) Test_FetchNextConversions_Success() {
	pspConversion := s.newPSPConversion()

	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_CONVERSIONS.String(), "1"),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextConversionsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextConversionsRequest) (*models.FetchNextConversionsResponse, error) {
		return &models.FetchNextConversionsResponse{
			Conversions: []models.PSPConversion{pspConversion},
			NewState:    []byte(`{}`),
			HasMore:     false,
		}, nil
	})
	s.env.OnActivity(activities.StorageConversionsUpsertActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, conversions []models.Conversion) error {
		s.Equal(1, len(conversions))
		s.Equal(pspConversion.Reference, conversions[0].Reference)
		return nil
	})
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchNextConversions, FetchNextConversions{
		ConnectorID: s.connectorID,
		FromPayload: &FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextConversions_EmptyResponse_Success() {
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_CONVERSIONS.String(), "1"),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextConversionsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextConversionsRequest) (*models.FetchNextConversionsResponse, error) {
		return &models.FetchNextConversionsResponse{
			Conversions: []models.PSPConversion{},
			NewState:    []byte(`{}`),
			HasMore:     false,
		}, nil
	})
	// No StorageConversionsUpsert call when empty
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchNextConversions, FetchNextConversions{
		ConnectorID: s.connectorID,
		FromPayload: &FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextConversions_StorageStatesGet_Error() {
	expectedErr := temporal.NewNonRetryableApplicationError("error-test", "STORAGE", errors.New("error-test"))
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(nil, expectedErr)

	s.env.ExecuteWorkflow(RunFetchNextConversions, FetchNextConversions{
		ConnectorID: s.connectorID,
		FromPayload: &FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_FetchNextConversions_PluginFetchNextConversions_Error() {
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_CONVERSIONS.String(), "1"),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	expectedErr := temporal.NewNonRetryableApplicationError("error-test", "PLUGIN", errors.New("error-test"))
	s.env.OnActivity(activities.PluginFetchNextConversionsActivity, mock.Anything, mock.Anything).Once().Return(nil, expectedErr)

	s.env.ExecuteWorkflow(RunFetchNextConversions, FetchNextConversions{
		ConnectorID: s.connectorID,
		FromPayload: &FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_FetchNextConversions_StorageConversionsUpsert_Error() {
	pspConversion := s.newPSPConversion()

	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_CONVERSIONS.String(), "1"),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextConversionsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextConversionsRequest) (*models.FetchNextConversionsResponse, error) {
		return &models.FetchNextConversionsResponse{
			Conversions: []models.PSPConversion{pspConversion},
			NewState:    []byte(`{}`),
			HasMore:     false,
		}, nil
	})
	expectedErr := temporal.NewNonRetryableApplicationError("error-test", "STORAGE", errors.New("error-test"))
	s.env.OnActivity(activities.StorageConversionsUpsertActivity, mock.Anything, mock.Anything).Once().Return(expectedErr)

	s.env.ExecuteWorkflow(RunFetchNextConversions, FetchNextConversions{
		ConnectorID: s.connectorID,
		FromPayload: &FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_FetchNextConversions_HasMore_Success() {
	pspConversion := s.newPSPConversion()
	pspConversion2 := s.newPSPConversion()
	pspConversion2.Reference = "test-conversion-2"

	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_CONVERSIONS.String(), "1"),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	// First call returns hasMore=true
	s.env.OnActivity(activities.PluginFetchNextConversionsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextConversionsRequest) (*models.FetchNextConversionsResponse, error) {
		return &models.FetchNextConversionsResponse{
			Conversions: []models.PSPConversion{pspConversion},
			NewState:    []byte(`{"cursor":"page2"}`),
			HasMore:     true,
		}, nil
	})
	s.env.OnActivity(activities.StorageConversionsUpsertActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	// Second call returns hasMore=false
	s.env.OnActivity(activities.PluginFetchNextConversionsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextConversionsRequest) (*models.FetchNextConversionsResponse, error) {
		return &models.FetchNextConversionsResponse{
			Conversions: []models.PSPConversion{pspConversion2},
			NewState:    []byte(`{}`),
			HasMore:     false,
		}, nil
	})
	s.env.OnActivity(activities.StorageConversionsUpsertActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchNextConversions, FetchNextConversions{
		ConnectorID: s.connectorID,
		FromPayload: &FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}
