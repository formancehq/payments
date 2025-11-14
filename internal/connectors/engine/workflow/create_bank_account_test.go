package workflow

import (
	"context"
	"errors"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_CreateBankAccount_Success() {
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

	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_SUCCEEDED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateBankAccount, CreateBankAccount{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID: s.connectorID,
		BankAccount: s.bankAccount,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_CreateBankAccount_PluginCreateBankAccount_Error() {
	s.env.OnActivity(activities.PluginCreateBankAccountActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
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
		ConnectorID: s.connectorID,
		BankAccount: s.bankAccount,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_CreateBankAccount_StorageAccountsStore_Error() {
	s.env.OnActivity(activities.PluginCreateBankAccountActivity, mock.Anything, mock.Anything).Once().Return(&models.CreateBankAccountResponse{
		RelatedAccount: s.pspAccount,
	}, nil)
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
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
		ConnectorID: s.connectorID,
		BankAccount: s.bankAccount,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_CreateBankAccount_StorageBankAccountsAddRelatedAccount_Error() {
	s.env.OnActivity(activities.PluginCreateBankAccountActivity, mock.Anything, mock.Anything).Once().Return(&models.CreateBankAccountResponse{
		RelatedAccount: s.pspAccount,
	}, nil)
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageBankAccountsAddRelatedAccountActivity, mock.Anything, s.bankAccount.ID, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
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
		ConnectorID: s.connectorID,
		BankAccount: s.bankAccount,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_CreateBankAccount_StorageTasksStore_Error() {
	s.env.OnActivity(activities.PluginCreateBankAccountActivity, mock.Anything, mock.Anything).Once().Return(&models.CreateBankAccountResponse{
		RelatedAccount: s.pspAccount,
	}, nil)
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageBankAccountsAddRelatedAccountActivity, mock.Anything, s.bankAccount.ID, mock.Anything).Once().Return(nil)
	// When StorageTasksStore fails in updateTask, it returns early
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		return temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test"))
	})

	s.env.ExecuteWorkflow(RunCreateBankAccount, CreateBankAccount{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID: s.connectorID,
		BankAccount: s.bankAccount,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err) // Workflow should fail when StorageTasksStore fails
	s.ErrorContains(err, "error-test")
}
