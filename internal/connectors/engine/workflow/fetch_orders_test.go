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

func (s *UnitTestSuite) newFetchPSPOrder() models.PSPOrder {
	return models.PSPOrder{
		Reference:           "test-order-" + s.connectorID.Reference.String()[:8],
		CreatedAt:           time.Now().UTC(),
		Direction:           models.ORDER_DIRECTION_BUY,
		SourceAsset:         "USD/2",
		TargetAsset:         "BTC/8",
		Type:                models.ORDER_TYPE_LIMIT,
		Status:              models.ORDER_STATUS_FILLED,
		BaseQuantityOrdered: big.NewInt(100000000),
		BaseQuantityFilled:  big.NewInt(100000000),
		LimitPrice:          big.NewInt(5000000000000),
		TimeInForce:         models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
		Metadata:            map[string]string{"key": "value"},
		Raw:                 []byte(`{}`),
	}
}

func (s *UnitTestSuite) Test_FetchNextOrders_Success() {
	pspOrder := s.newFetchPSPOrder()

	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_ORDERS.String(), "1"),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextOrdersActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextOrdersRequest) (*models.FetchNextOrdersResponse, error) {
		return &models.FetchNextOrdersResponse{
			Orders:   []models.PSPOrder{pspOrder},
			NewState: []byte(`{}`),
			HasMore:  false,
		}, nil
	})
	s.env.OnActivity(activities.StorageOrdersUpsertActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, orders []models.Order) error {
		s.Equal(1, len(orders))
		s.Equal(pspOrder.Reference, orders[0].Reference)
		return nil
	})
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchNextOrders, FetchNextOrders{
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

func (s *UnitTestSuite) Test_FetchNextOrders_EmptyResponse_Success() {
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_ORDERS.String(), "1"),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextOrdersActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextOrdersRequest) (*models.FetchNextOrdersResponse, error) {
		return &models.FetchNextOrdersResponse{
			Orders:   []models.PSPOrder{},
			NewState: []byte(`{}`),
			HasMore:  false,
		}, nil
	})
	// No StorageOrdersUpsert call when empty
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchNextOrders, FetchNextOrders{
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

func (s *UnitTestSuite) Test_FetchNextOrders_StorageStatesGet_Error() {
	expectedErr := temporal.NewNonRetryableApplicationError("error-test", "STORAGE", errors.New("error-test"))
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(nil, expectedErr)

	s.env.ExecuteWorkflow(RunFetchNextOrders, FetchNextOrders{
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

func (s *UnitTestSuite) Test_FetchNextOrders_PluginFetchNextOrders_Error() {
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_ORDERS.String(), "1"),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	expectedErr := temporal.NewNonRetryableApplicationError("error-test", "PLUGIN", errors.New("error-test"))
	s.env.OnActivity(activities.PluginFetchNextOrdersActivity, mock.Anything, mock.Anything).Once().Return(nil, expectedErr)

	s.env.ExecuteWorkflow(RunFetchNextOrders, FetchNextOrders{
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

func (s *UnitTestSuite) Test_FetchNextOrders_StorageOrdersUpsert_Error() {
	pspOrder := s.newFetchPSPOrder()

	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_ORDERS.String(), "1"),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextOrdersActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextOrdersRequest) (*models.FetchNextOrdersResponse, error) {
		return &models.FetchNextOrdersResponse{
			Orders:   []models.PSPOrder{pspOrder},
			NewState: []byte(`{}`),
			HasMore:  false,
		}, nil
	})
	expectedErr := temporal.NewNonRetryableApplicationError("error-test", "STORAGE", errors.New("error-test"))
	s.env.OnActivity(activities.StorageOrdersUpsertActivity, mock.Anything, mock.Anything).Once().Return(expectedErr)

	s.env.ExecuteWorkflow(RunFetchNextOrders, FetchNextOrders{
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

func (s *UnitTestSuite) Test_FetchNextOrders_HasMore_Success() {
	pspOrder := s.newFetchPSPOrder()
	pspOrder2 := s.newFetchPSPOrder()
	pspOrder2.Reference = "test-order-2"

	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_ORDERS.String(), "1"),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	// First call returns hasMore=true
	s.env.OnActivity(activities.PluginFetchNextOrdersActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextOrdersRequest) (*models.FetchNextOrdersResponse, error) {
		return &models.FetchNextOrdersResponse{
			Orders:   []models.PSPOrder{pspOrder},
			NewState: []byte(`{"cursor":"page2"}`),
			HasMore:  true,
		}, nil
	})
	s.env.OnActivity(activities.StorageOrdersUpsertActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	// Second call returns hasMore=false
	s.env.OnActivity(activities.PluginFetchNextOrdersActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextOrdersRequest) (*models.FetchNextOrdersResponse, error) {
		return &models.FetchNextOrdersResponse{
			Orders:   []models.PSPOrder{pspOrder2},
			NewState: []byte(`{}`),
			HasMore:  false,
		}, nil
	})
	s.env.OnActivity(activities.StorageOrdersUpsertActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchNextOrders, FetchNextOrders{
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
