package workflow

import (
	"context"
	"errors"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_InstallConnector_Success() {
	s.env.OnActivity(activities.PluginInstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.InstallConnectorRequest) (*models.InstallResponse, error) {
		return &models.InstallResponse{
			Workflow: []models.ConnectorTaskTree{},
		}, nil
	})
	s.env.OnActivity(activities.StorageConnectorTasksTreeStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TasksTreeStoreRequest) error {
		return nil
	})
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).Once().Return(
		&s.connector,
		nil,
	)
	s.env.OnWorkflow(Run, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunInstallConnector, InstallConnector{
		ConnectorID: s.connectorID,
		Config:      models.DefaultConfig(),
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_InstallConnector_ConnectorScheduledForDeletion_Success() {
	connector := s.connector
	connector.ScheduledForDeletion = true
	s.env.OnActivity(activities.PluginInstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.InstallConnectorRequest) (*models.InstallResponse, error) {
		return &models.InstallResponse{
			Workflow: []models.ConnectorTaskTree{},
		}, nil
	})
	s.env.OnActivity(activities.StorageConnectorTasksTreeStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TasksTreeStoreRequest) error {
		return nil
	})
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).Once().Return(
		&connector,
		nil,
	)

	s.env.ExecuteWorkflow(RunInstallConnector, InstallConnector{
		ConnectorID: s.connectorID,
		Config:      models.DefaultConfig(),
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_InstallConnector_NoConfigs_Success() {
	s.env.OnActivity(activities.PluginInstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.InstallConnectorRequest) (*models.InstallResponse, error) {
		return &models.InstallResponse{
			Workflow: []models.ConnectorTaskTree{},
		}, nil
	})
	s.env.OnActivity(activities.StorageConnectorTasksTreeStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TasksTreeStoreRequest) error {
		return nil
	})
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).Once().Return(
		&s.connector,
		nil,
	)
	s.env.OnWorkflow(Run, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunInstallConnector, InstallConnector{
		ConnectorID: s.connectorID,
		Config:      models.DefaultConfig(),
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_InstallConnector_PluginInstallConnector_Error() {
	s.env.OnActivity(activities.PluginInstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "PLUGIN", errors.New("error-test")),
	)
	s.env.OnActivity(activities.StorageConnectorsDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)

	s.env.ExecuteWorkflow(RunInstallConnector, InstallConnector{
		ConnectorID: s.connectorID,
		Config:      models.DefaultConfig(),
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_InstallConnector_StorageConnectorTasksTreeStore_Error() {
	s.env.OnActivity(activities.PluginInstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(
		&models.InstallResponse{
			Workflow: []models.ConnectorTaskTree{},
		}, nil)
	s.env.OnActivity(activities.StorageConnectorTasksTreeStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("error-test")),
	)
	s.env.OnActivity(activities.StorageConnectorsDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)

	s.env.ExecuteWorkflow(RunInstallConnector, InstallConnector{
		ConnectorID: s.connectorID,
		Config:      models.DefaultConfig(),
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_InstallConnector_StorageConnectorsDelete_Error() {
	s.env.OnActivity(activities.PluginInstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(&models.InstallResponse{
		Workflow: []models.ConnectorTaskTree{},
	}, nil)
	s.env.OnActivity(activities.StorageConnectorTasksTreeStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorsDeleteActivity, mock.Anything, s.connectorID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "STORAGE", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunInstallConnector, InstallConnector{
		ConnectorID: s.connectorID,
		Config:      models.DefaultConfig(),
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_InstallConnector_StorageConnectorsGet_Error() {
	s.env.OnActivity(activities.PluginInstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(&models.InstallResponse{
		Workflow: []models.ConnectorTaskTree{},
	}, nil)
	s.env.OnActivity(activities.StorageConnectorTasksTreeStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorsDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("error-test", "STORAGE", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunInstallConnector, InstallConnector{
		ConnectorID: s.connectorID,
		Config:      models.DefaultConfig(),
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_InstallConnector_Run_Error() {
	s.env.OnActivity(activities.PluginInstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(&models.InstallResponse{
		Workflow: []models.ConnectorTaskTree{},
	}, nil)
	s.env.OnActivity(activities.StorageConnectorTasksTreeStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).Once().Return(
		&s.connector,
		nil,
	)
	s.env.OnWorkflow(Run, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunInstallConnector, InstallConnector{
		ConnectorID: s.connectorID,
		Config:      models.DefaultConfig(),
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	// We only check that the workflow has started, we don't check if it has completed
	// without error
	s.NoError(err)
}

func (s *UnitTestSuite) Test_InstallConnector_Run_ErrorAlreadyStarted() {
	s.env.OnActivity(activities.PluginInstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(&models.InstallResponse{
		Workflow: []models.ConnectorTaskTree{},
	}, nil)
	s.env.OnActivity(activities.StorageConnectorTasksTreeStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).Once().Return(
		&s.connector,
		nil,
	)
	s.env.OnWorkflow(Run, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		serviceerror.NewWorkflowExecutionAlreadyStarted("test", "test", "test"),
	)

	s.env.ExecuteWorkflow(RunInstallConnector, InstallConnector{
		ConnectorID: s.connectorID,
		Config:      models.DefaultConfig(),
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	// We only check that the workflow has started, we don't check if it has completed
	// without error
	s.NoError(err)
}
