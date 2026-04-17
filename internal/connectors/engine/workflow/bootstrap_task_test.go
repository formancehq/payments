package workflow

import (
	"context"
	"errors"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
)

func (s *UnitTestSuite) Test_BootstrapTask_SinglePage_Success() {
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_ACCOUNTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextAccountsActivity, mock.Anything, mock.Anything).Once().Return(
		&models.FetchNextAccountsResponse{
			Accounts: []models.PSPAccount{s.pspAccount},
			NewState: []byte(`{"cursor":""}`),
			HasMore:  false,
		},
		nil,
	)
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(
		func(ctx context.Context, state models.State) error {
			s.Equal([]byte(`{"cursor":""}`), []byte(state.State))
			return nil
		},
	)

	s.env.ExecuteWorkflow(RunBootstrapTask, BootstrapTaskRequest{
		ConnectorID: s.connectorID,
		TaskType:    models.TASK_FETCH_ACCOUNTS,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_BootstrapTask_MultiPage_PerPageStatePersisted() {
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_ACCOUNTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{"page":0}`),
		},
		nil,
	)

	// Three pages: first two with HasMore=true, last with HasMore=false.
	page := 0
	s.env.OnActivity(activities.PluginFetchNextAccountsActivity, mock.Anything, mock.Anything).Times(3).Return(
		func(ctx context.Context, req activities.FetchNextAccountsRequest) (*models.FetchNextAccountsResponse, error) {
			page++
			return &models.FetchNextAccountsResponse{
				Accounts: []models.PSPAccount{s.pspAccount},
				NewState: []byte(fmt.Sprintf(`{"page":%d}`, page)),
				HasMore:  page < 3,
			}, nil
		},
	)

	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Times(3).Return(nil)

	// State is persisted after every page — three times.
	stored := 0
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Times(3).Return(
		func(ctx context.Context, state models.State) error {
			stored++
			s.Equal([]byte(fmt.Sprintf(`{"page":%d}`, stored)), []byte(state.State))
			return nil
		},
	)

	s.env.ExecuteWorkflow(RunBootstrapTask, BootstrapTaskRequest{
		ConnectorID: s.connectorID,
		TaskType:    models.TASK_FETCH_ACCOUNTS,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
	s.Equal(3, stored)
}

func (s *UnitTestSuite) Test_BootstrapTask_EmptyPage_NoStorageAccountsStoreCall() {
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_ACCOUNTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextAccountsActivity, mock.Anything, mock.Anything).Once().Return(
		&models.FetchNextAccountsResponse{
			Accounts: []models.PSPAccount{},
			NewState: []byte(`{}`),
			HasMore:  false,
		},
		nil,
	)
	// StorageAccountsStore must not be called when the page is empty.
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunBootstrapTask, BootstrapTaskRequest{
		ConnectorID: s.connectorID,
		TaskType:    models.TASK_FETCH_ACCOUNTS,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_BootstrapTask_UnrecognizedTaskType_NonRetryableError() {
	s.env.ExecuteWorkflow(RunBootstrapTask, BootstrapTaskRequest{
		ConnectorID: s.connectorID,
		TaskType:    models.TaskType(9999),
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.Contains(err.Error(), "bootstrap does not support task type")
}

func (s *UnitTestSuite) Test_BootstrapTask_PluginError_Propagates() {
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_ACCOUNTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextAccountsActivity, mock.Anything, mock.Anything).Once().Return(
		(*models.FetchNextAccountsResponse)(nil),
		errors.New("upstream API failure"),
	)

	s.env.ExecuteWorkflow(RunBootstrapTask, BootstrapTaskRequest{
		ConnectorID: s.connectorID,
		TaskType:    models.TASK_FETCH_ACCOUNTS,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}
