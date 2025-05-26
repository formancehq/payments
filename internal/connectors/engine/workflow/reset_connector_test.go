package workflow

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func (s *UnitTestSuite) Test_ResetConnector_Success() {
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).Once().Return(
		&s.connector,
		nil,
	)
	s.env.OnActivity(activities.StorageConnectorsScheduleForDeletionActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnWorkflow(RunUninstallConnector, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req UninstallConnector) error {
		s.Equal(s.connectorID, req.ConnectorID)
		return nil
	})
	s.env.OnActivity(activities.StorageConnectorsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, connector models.Connector) error {
		s.NotEqual(s.connectorID, connector.ID)
		s.Equal(s.connector.Name, connector.Name)
		return nil
	})
	s.env.OnWorkflow(RunInstallConnector, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, sendEvents SendEvents) error {
		s.Nil(sendEvents.Balance)
		s.Nil(sendEvents.Account)
		s.NotNil(sendEvents.ConnectorReset)
		s.Equal(s.connectorID, *sendEvents.ConnectorReset)
		s.Nil(sendEvents.Payment)
		s.Nil(sendEvents.PoolsCreation)
		s.Nil(sendEvents.PoolsDeletion)
		s.Nil(sendEvents.BankAccount)
		return nil
	})
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_SUCCEEDED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunResetConnector, ResetConnector{
		ConnectorID:       s.connectorID,
		DefaultWorkerName: "test",
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_ResetConnector_StorageConnectorsGet_Error() {
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("error-test", "STORAGE", fmt.Errorf("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunResetConnector, ResetConnector{
		ConnectorID:       s.connectorID,
		DefaultWorkerName: "test",
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_ResetConnector_StorageConnectorsScheduleForDeletion_Error() {
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).Once().Return(
		&s.connector,
		nil,
	)
	s.env.OnActivity(activities.StorageConnectorsScheduleForDeletionActivity, mock.Anything, s.connectorID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "STORAGE", fmt.Errorf("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunResetConnector, ResetConnector{
		ConnectorID:       s.connectorID,
		DefaultWorkerName: "test",
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_ResetConnector_RunUninstallConnector_Error() {
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).Once().Return(
		&s.connector,
		nil,
	)
	s.env.OnActivity(activities.StorageConnectorsScheduleForDeletionActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnWorkflow(RunUninstallConnector, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "WORKFLOW", fmt.Errorf("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunResetConnector, ResetConnector{
		ConnectorID:       s.connectorID,
		DefaultWorkerName: "test",
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_ResetConnector_StorageConnectorsStore_Error() {
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).Once().Return(
		&s.connector,
		nil,
	)
	s.env.OnActivity(activities.StorageConnectorsScheduleForDeletionActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnWorkflow(RunUninstallConnector, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorsStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "STORAGE", fmt.Errorf("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunResetConnector, ResetConnector{
		ConnectorID:       s.connectorID,
		DefaultWorkerName: "test",
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_ResetConnector_RunInstallConnector_Error() {
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).Once().Return(
		&s.connector,
		nil,
	)
	s.env.OnActivity(activities.StorageConnectorsScheduleForDeletionActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnWorkflow(RunUninstallConnector, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunInstallConnector, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "STORAGE", fmt.Errorf("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunResetConnector, ResetConnector{
		ConnectorID:       s.connectorID,
		DefaultWorkerName: "test",
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_ResetConnector_RunSendEvents_Error() {
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).Once().Return(
		&s.connector,
		nil,
	)
	s.env.OnActivity(activities.StorageConnectorsScheduleForDeletionActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnWorkflow(RunUninstallConnector, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunInstallConnector, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "STORAGE", fmt.Errorf("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunResetConnector, ResetConnector{
		ConnectorID:       s.connectorID,
		DefaultWorkerName: "test",
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_ResetConnector_StorageTasksStoreActivity_Error() {
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, s.connectorID).Once().Return(
		&s.connector,
		nil,
	)
	s.env.OnActivity(activities.StorageConnectorsScheduleForDeletionActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnWorkflow(RunUninstallConnector, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunInstallConnector, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "STORAGE", fmt.Errorf("error-test")),
	)

	s.env.ExecuteWorkflow(RunResetConnector, ResetConnector{
		ConnectorID:       s.connectorID,
		DefaultWorkerName: "test",
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}
