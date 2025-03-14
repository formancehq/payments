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

func (s *UnitTestSuite) Test_CreateCounterParty_Success() {
	s.env.OnActivity(activities.StorageCounterPartiesGetActivity, mock.Anything, s.counterParty.ID).Once().Return(&s.counterParty, nil)
	s.env.OnActivity(activities.StorageBankAccountsGetActivity, mock.Anything, s.bankAccount.ID, true).Once().Return(&s.bankAccount, nil)
	s.env.OnActivity(activities.PluginCreateBankAccountActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, request activities.CreateBankAccountRequest) (*models.CreateBankAccountResponse, error) {
		s.Equal(s.connectorID, request.ConnectorID)
		s.Equal(s.counterParty.ID, request.Req.CounterParty.ID)
		s.Equal(s.bankAccount.ID, request.Req.CounterParty.BankAccount.ID)
		return &models.CreateBankAccountResponse{
			RelatedAccount: s.pspAccount,
		}, nil
	})
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, accounts []models.Account) error {
		s.Equal(1, len(accounts))
		s.Equal(s.accountID, accounts[0].ID)
		return nil
	})
	s.env.OnActivity(activities.StorageCounterPartiesAddRelatedAccountActivity, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, counterPartyID uuid.UUID, relatedAccount models.CounterPartiesRelatedAccount) error {
		s.Equal(s.counterParty.ID, counterPartyID)
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
		s.Nil(sendEvents.BankAccount)
		s.NotNil(sendEvents.CounterParty)
		s.Equal(s.counterParty.ID, sendEvents.CounterParty.ID)
		return nil
	})
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_SUCCEEDED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateCounterParty, CreateCounterParty{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:    s.connectorID,
		CounterPartyID: s.counterParty.ID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_CreateCounterParty_StorageCounterPartyGet_Error() {
	s.env.OnActivity(activities.StorageCounterPartiesGetActivity, mock.Anything, s.counterParty.ID).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "test", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateCounterParty, CreateCounterParty{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:    s.connectorID,
		CounterPartyID: s.counterParty.ID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreateCounterParty_StorageBankAccountsGet_Error() {
	s.env.OnActivity(activities.StorageCounterPartiesGetActivity, mock.Anything, s.counterParty.ID).Once().Return(&s.counterParty, nil)
	s.env.OnActivity(activities.StorageBankAccountsGetActivity, mock.Anything, s.bankAccount.ID, true).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "test", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateCounterParty, CreateCounterParty{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:    s.connectorID,
		CounterPartyID: s.counterParty.ID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreateCounterParty_PluginCreateBankAccount_Error() {
	s.env.OnActivity(activities.StorageCounterPartiesGetActivity, mock.Anything, s.counterParty.ID).Once().Return(&s.counterParty, nil)
	s.env.OnActivity(activities.StorageBankAccountsGetActivity, mock.Anything, s.bankAccount.ID, true).Once().Return(&s.bankAccount, nil)
	s.env.OnActivity(activities.PluginCreateBankAccountActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "test", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateCounterParty, CreateCounterParty{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:    s.connectorID,
		CounterPartyID: s.counterParty.ID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreateCounterParty_StorageAccountsStore_Error() {
	s.env.OnActivity(activities.StorageCounterPartiesGetActivity, mock.Anything, s.counterParty.ID).Once().Return(&s.counterParty, nil)
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

	s.env.ExecuteWorkflow(RunCreateCounterParty, CreateCounterParty{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:    s.connectorID,
		CounterPartyID: s.counterParty.ID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreateCounterParty_StorageCounterPartiesAddRelatedAccount_Error() {
	s.env.OnActivity(activities.StorageCounterPartiesGetActivity, mock.Anything, s.counterParty.ID).Once().Return(&s.counterParty, nil)
	s.env.OnActivity(activities.StorageBankAccountsGetActivity, mock.Anything, s.bankAccount.ID, true).Once().Return(&s.bankAccount, nil)
	s.env.OnActivity(activities.PluginCreateBankAccountActivity, mock.Anything, mock.Anything).Once().Return(&models.CreateBankAccountResponse{
		RelatedAccount: s.pspAccount,
	}, nil)
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageCounterPartiesAddRelatedAccountActivity, mock.Anything, s.counterParty.ID, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "test", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateCounterParty, CreateCounterParty{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:    s.connectorID,
		CounterPartyID: s.counterParty.ID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreateCounterParty_RunSendEvents_Error() {
	s.env.OnActivity(activities.StorageCounterPartiesGetActivity, mock.Anything, s.counterParty.ID).Once().Return(&s.counterParty, nil)
	s.env.OnActivity(activities.StorageBankAccountsGetActivity, mock.Anything, s.bankAccount.ID, true).Once().Return(&s.bankAccount, nil)
	s.env.OnActivity(activities.PluginCreateBankAccountActivity, mock.Anything, mock.Anything).Once().Return(&models.CreateBankAccountResponse{
		RelatedAccount: s.pspAccount,
	}, nil)
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageCounterPartiesAddRelatedAccountActivity, mock.Anything, s.counterParty.ID, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(s.w.runSendEvents, mock.Anything, mock.Anything).Once().Return(temporal.NewNonRetryableApplicationError("test", "test", errors.New("test")))
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return nil
	})

	s.env.ExecuteWorkflow(RunCreateCounterParty, CreateCounterParty{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:    s.connectorID,
		CounterPartyID: s.counterParty.ID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreateCounterParty_StorageTasksStore_Error() {
	s.env.OnActivity(activities.StorageCounterPartiesGetActivity, mock.Anything, s.counterParty.ID).Once().Return(&s.counterParty, nil)
	s.env.OnActivity(activities.StorageBankAccountsGetActivity, mock.Anything, s.bankAccount.ID, true).Once().Return(&s.bankAccount, nil)
	s.env.OnActivity(activities.PluginCreateBankAccountActivity, mock.Anything, mock.Anything).Once().Return(&models.CreateBankAccountResponse{
		RelatedAccount: s.pspAccount,
	}, nil)
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageCounterPartiesAddRelatedAccountActivity, mock.Anything, s.counterParty.ID, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(s.w.runSendEvents, mock.Anything, mock.Anything).Once().Return(temporal.NewNonRetryableApplicationError("test", "test", errors.New("test")))
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, task models.Task) error {
		s.Equal(models.TASK_STATUS_FAILED, task.Status)
		return temporal.NewNonRetryableApplicationError("test", "test", errors.New("test"))
	})

	s.env.ExecuteWorkflow(RunCreateCounterParty, CreateCounterParty{
		TaskID: models.TaskID{
			Reference:   "test",
			ConnectorID: s.connectorID,
		},
		ConnectorID:    s.connectorID,
		CounterPartyID: s.counterParty.ID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "test")
}
