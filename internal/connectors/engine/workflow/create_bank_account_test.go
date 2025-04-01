package workflow

import (
	"context"
	"errors"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func (s *UnitTestSuite) Test_CreateBankAccount_Success() {
	s.env.OnActivity(activities.StorageBankAccountsGetActivity, mock.Anything, s.bankAccount.ID, true).Once().Return(&s.bankAccount, nil)
	s.env.OnActivity(activities.PluginCreateBankAccountActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, request activities.CreateBankAccountRequest) (*models.CreateBankAccountResponse, error) {
		s.Equal(s.connectorID, request.ConnectorID)
		s.Equal(s.bankAccount.ID, request.Req.BankAccount.ID)
		return &models.CreateBankAccountResponse{
			RelatedAccount: s.pspAccount,
		}, nil
	})
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, accounts []models.Account) error {
		s.Equal(1, len(accounts))
		s.Equal(s.accountID, accounts[0].ID)
		return nil
	})
	s.env.OnActivity(activities.StorageBankAccountsAddRelatedAccountActivity, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, bankAccountID uuid.UUID, relatedAccount models.BankAccountRelatedAccount) error {
		s.Equal(s.bankAccount.ID, bankAccountID)
		s.Equal(s.accountID, relatedAccount.AccountID)
		return nil
	})
	s.env.OnWorkflow(s.w.runSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, sendEvents SendEvents) error {
		s.Nil(sendEvents.Balance)
		s.Nil(sendEvents.Account)
		s.Nil(sendEvents.ConnectorReset)
		s.Nil(sendEvents.Payment)
		s.Nil(sendEvents.PoolsCreation)
		s.Nil(sendEvents.PoolsDeletion)
		s.NotNil(sendEvents.BankAccount)
		s.Equal(s.bankAccount.ID, sendEvents.BankAccount.ID)
		return nil
	})
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_SUCCEEDED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateBankAccount, CreateBankAccount{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:   s.connectorID,
		BankAccountID: s.bankAccount.ID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_CreateBankAccount_StorageBankAccountGet_Error() {
	s.env.OnActivity(activities.StorageBankAccountsGetActivity, mock.Anything, s.bankAccount.ID, true).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "test", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateBankAccount, CreateBankAccount{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:   s.connectorID,
		BankAccountID: s.bankAccount.ID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreateBankAccount_PluginCreateBankAccount_Error() {
	s.env.OnActivity(activities.StorageBankAccountsGetActivity, mock.Anything, s.bankAccount.ID, true).Once().Return(&s.bankAccount, nil)
	s.env.OnActivity(activities.PluginCreateBankAccountActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "test", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateBankAccount, CreateBankAccount{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:   s.connectorID,
		BankAccountID: s.bankAccount.ID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreateBankAccount_StorageAccountsStore_Error() {
	s.env.OnActivity(activities.StorageBankAccountsGetActivity, mock.Anything, s.bankAccount.ID, true).Once().Return(&s.bankAccount, nil)
	s.env.OnActivity(activities.PluginCreateBankAccountActivity, mock.Anything, mock.Anything).Once().Return(&models.CreateBankAccountResponse{
		RelatedAccount: s.pspAccount,
	}, nil)
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "test", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateBankAccount, CreateBankAccount{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:   s.connectorID,
		BankAccountID: s.bankAccount.ID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreateBankAccount_StorageBankAccountsAddRelatedAccount_Error() {
	s.env.OnActivity(activities.StorageBankAccountsGetActivity, mock.Anything, s.bankAccount.ID, true).Once().Return(&s.bankAccount, nil)
	s.env.OnActivity(activities.PluginCreateBankAccountActivity, mock.Anything, mock.Anything).Once().Return(&models.CreateBankAccountResponse{
		RelatedAccount: s.pspAccount,
	}, nil)
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageBankAccountsAddRelatedAccountActivity, mock.Anything, s.bankAccount.ID, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "test", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateBankAccount, CreateBankAccount{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:   s.connectorID,
		BankAccountID: s.bankAccount.ID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreateBankAccount_StorageTasksStore_Error() {
	s.env.OnActivity(activities.StorageBankAccountsGetActivity, mock.Anything, s.bankAccount.ID, true).Once().Return(&s.bankAccount, nil)
	s.env.OnActivity(activities.PluginCreateBankAccountActivity, mock.Anything, mock.Anything).Once().Return(&models.CreateBankAccountResponse{
		RelatedAccount: s.pspAccount,
	}, nil)
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageBankAccountsAddRelatedAccountActivity, mock.Anything, s.bankAccount.ID, mock.Anything).Once().Return(temporal.NewNonRetryableApplicationError("test", "test", errors.New("test")))
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return temporal.NewNonRetryableApplicationError("test", "test", errors.New("test"))
	})

	s.env.ExecuteWorkflow(RunCreateBankAccount, CreateBankAccount{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:   s.connectorID,
		BankAccountID: s.bankAccount.ID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "test")
}
