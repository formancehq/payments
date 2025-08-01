package workflow

import (
	"context"
	"errors"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func (s *UnitTestSuite) Test_UninstallConnector_WithoutTaskID_Success() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return([]models.WebhookConfig{}, nil)
	s.env.OnWorkflow(RunTerminateSchedules, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunTerminateWorkflows, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginUninstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(nil, nil)
	s.env.OnActivity(activities.StorageEventsSentDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorTasksTreeDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageBankAccountsDeleteRelatedAccountsActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageWebhooksConfigsDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageWebhooksDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StoragePoolsRemoveAccountsFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorsDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)

	s.env.ExecuteWorkflow(RunUninstallConnector, UninstallConnector{
		ConnectorID: s.connectorID,
		TaskID:      nil,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_UninstallConnector_WithTaskID_Success() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return([]models.WebhookConfig{}, nil)
	s.env.OnWorkflow(RunTerminateSchedules, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunTerminateWorkflows, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginUninstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(nil, nil)
	s.env.OnActivity(activities.StorageEventsSentDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorTasksTreeDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageBankAccountsDeleteRelatedAccountsActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageWebhooksConfigsDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageWebhooksDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StoragePoolsRemoveAccountsFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorsDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_SUCCEEDED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunUninstallConnector, UninstallConnector{
		ConnectorID: s.connectorID,
		TaskID: &models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_UninstallConnector_RunTerminateSchedules_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return([]models.WebhookConfig{}, nil)
	s.env.OnWorkflow(RunTerminateSchedules, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunUninstallConnector, UninstallConnector{
		ConnectorID: s.connectorID,
		TaskID: &models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_UninstallConnector_PluginUninstallConnector_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return([]models.WebhookConfig{}, nil)
	s.env.OnWorkflow(RunTerminateSchedules, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunTerminateWorkflows, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginUninstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(nil,
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")))
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunUninstallConnector, UninstallConnector{
		ConnectorID: s.connectorID,
		TaskID: &models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_UninstallConnector_StorageEventsSentDelete_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return([]models.WebhookConfig{}, nil)
	s.env.OnWorkflow(RunTerminateSchedules, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunTerminateWorkflows, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginUninstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(nil, nil)
	s.env.OnActivity(activities.StorageEventsSentDeleteActivity, mock.Anything, s.connectorID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunUninstallConnector, UninstallConnector{
		ConnectorID: s.connectorID,
		TaskID: &models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_UninstallConnector_StorageSchedulesDeleteFromConnectorID_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return([]models.WebhookConfig{}, nil)
	s.env.OnWorkflow(RunTerminateSchedules, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunTerminateWorkflows, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginUninstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(nil, nil)
	s.env.OnActivity(activities.StorageEventsSentDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunUninstallConnector, UninstallConnector{
		ConnectorID: s.connectorID,
		TaskID: &models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_UninstallConnector_StorageInstancesDelete_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return([]models.WebhookConfig{}, nil)
	s.env.OnWorkflow(RunTerminateSchedules, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunTerminateWorkflows, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginUninstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(nil, nil)
	s.env.OnActivity(activities.StorageEventsSentDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesDeleteActivity, mock.Anything, s.connectorID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunUninstallConnector, UninstallConnector{
		ConnectorID: s.connectorID,
		TaskID: &models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_UninstallConnector_StorageConnectorTasksTreeDelete_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return([]models.WebhookConfig{}, nil)
	s.env.OnWorkflow(RunTerminateSchedules, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunTerminateWorkflows, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginUninstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(nil, nil)
	s.env.OnActivity(activities.StorageEventsSentDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorTasksTreeDeleteActivity, mock.Anything, s.connectorID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunUninstallConnector, UninstallConnector{
		ConnectorID: s.connectorID,
		TaskID: &models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_UninstallConnector_StorageTasksDeleteFromConnectorID_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return([]models.WebhookConfig{}, nil)
	s.env.OnWorkflow(RunTerminateSchedules, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunTerminateWorkflows, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginUninstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(nil, nil)
	s.env.OnActivity(activities.StorageEventsSentDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorTasksTreeDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunUninstallConnector, UninstallConnector{
		ConnectorID: s.connectorID,
		TaskID: &models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_UninstallConnector_StorageBankAccountsDeleteRelatedAccounts_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return([]models.WebhookConfig{}, nil)
	s.env.OnWorkflow(RunTerminateSchedules, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunTerminateWorkflows, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginUninstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(nil, nil)
	s.env.OnActivity(activities.StorageEventsSentDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorTasksTreeDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageBankAccountsDeleteRelatedAccountsActivity, mock.Anything, s.connectorID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunUninstallConnector, UninstallConnector{
		ConnectorID: s.connectorID,
		TaskID: &models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_UninstallConnector_StorageAccountsDelete_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return([]models.WebhookConfig{}, nil)
	s.env.OnWorkflow(RunTerminateSchedules, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunTerminateWorkflows, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginUninstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(nil, nil)
	s.env.OnActivity(activities.StorageEventsSentDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorTasksTreeDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageBankAccountsDeleteRelatedAccountsActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunUninstallConnector, UninstallConnector{
		ConnectorID: s.connectorID,
		TaskID: &models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_UninstallConnector_StoragePaymentsDelete_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return([]models.WebhookConfig{}, nil)
	s.env.OnWorkflow(RunTerminateSchedules, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunTerminateWorkflows, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginUninstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(nil, nil)
	s.env.OnActivity(activities.StorageEventsSentDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorTasksTreeDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageBankAccountsDeleteRelatedAccountsActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunUninstallConnector, UninstallConnector{
		ConnectorID: s.connectorID,
		TaskID: &models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_UninstallConnector_StorageStatesDelete_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return([]models.WebhookConfig{}, nil)
	s.env.OnWorkflow(RunTerminateSchedules, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunTerminateWorkflows, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginUninstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(nil, nil)
	s.env.OnActivity(activities.StorageEventsSentDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorTasksTreeDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageBankAccountsDeleteRelatedAccountsActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesDeleteActivity, mock.Anything, s.connectorID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunUninstallConnector, UninstallConnector{
		ConnectorID: s.connectorID,
		TaskID: &models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_UninstallConnector_StorageWebhooksConfigsDelete_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return([]models.WebhookConfig{}, nil)
	s.env.OnWorkflow(RunTerminateSchedules, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunTerminateWorkflows, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginUninstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(nil, nil)
	s.env.OnActivity(activities.StorageEventsSentDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorTasksTreeDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageBankAccountsDeleteRelatedAccountsActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageWebhooksConfigsDeleteActivity, mock.Anything, s.connectorID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunUninstallConnector, UninstallConnector{
		ConnectorID: s.connectorID,
		TaskID: &models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_UninstallConnector_StorageWebhooksDelete_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return([]models.WebhookConfig{}, nil)
	s.env.OnWorkflow(RunTerminateSchedules, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunTerminateWorkflows, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginUninstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(nil, nil)
	s.env.OnActivity(activities.StorageEventsSentDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorTasksTreeDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageBankAccountsDeleteRelatedAccountsActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageWebhooksConfigsDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageWebhooksDeleteActivity, mock.Anything, s.connectorID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunUninstallConnector, UninstallConnector{
		ConnectorID: s.connectorID,
		TaskID: &models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_UninstallConnector_StoragePoolsRemoveAccountsFromConnectorID_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return([]models.WebhookConfig{}, nil)
	s.env.OnWorkflow(RunTerminateSchedules, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunTerminateWorkflows, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginUninstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(nil, nil)
	s.env.OnActivity(activities.StorageEventsSentDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorTasksTreeDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageBankAccountsDeleteRelatedAccountsActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageWebhooksConfigsDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageWebhooksDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StoragePoolsRemoveAccountsFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunUninstallConnector, UninstallConnector{
		ConnectorID: s.connectorID,
		TaskID: &models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_UninstallConnector_StorageConnectorsDelete_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return([]models.WebhookConfig{}, nil)
	s.env.OnWorkflow(RunTerminateSchedules, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunTerminateWorkflows, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginUninstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(nil, nil)
	s.env.OnActivity(activities.StorageEventsSentDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorTasksTreeDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageBankAccountsDeleteRelatedAccountsActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageWebhooksConfigsDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageWebhooksDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StoragePoolsRemoveAccountsFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorsDeleteActivity, mock.Anything, s.connectorID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.Task)
		return nil
	})

	s.env.ExecuteWorkflow(RunUninstallConnector, UninstallConnector{
		ConnectorID: s.connectorID,
		TaskID: &models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_UninstallConnector_StorageTasksStore_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return([]models.WebhookConfig{}, nil)
	s.env.OnWorkflow(RunTerminateSchedules, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunTerminateWorkflows, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginUninstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(nil, nil)
	s.env.OnActivity(activities.StorageEventsSentDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageSchedulesDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorTasksTreeDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageBankAccountsDeleteRelatedAccountsActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageAccountsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentsDeleteFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageWebhooksConfigsDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageWebhooksDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StoragePoolsRemoveAccountsFromConnectorIDActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorsDeleteActivity, mock.Anything, s.connectorID).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunUninstallConnector, UninstallConnector{
		ConnectorID: s.connectorID,
		TaskID: &models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}
