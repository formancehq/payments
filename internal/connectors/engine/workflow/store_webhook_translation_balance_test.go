package workflow

import (
	"context"
	"errors"
	"time"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

// Tests for storeWebhookTranslation when receiving balance

func (s *UnitTestSuite) Test_StoreWebhookTranslation_Balance_Success() {
	// Prepare an account matching the PSP balance AccountReference
	acc := s.account
	acc.ID = s.accountID // ensure reference matches s.pspBalance.AccountReference ("test")
	acc.Reference = s.accountID.Reference

	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, models.AccountID{
		Reference:   s.pspBalance.AccountReference,
		ConnectorID: s.connectorID,
	}).Once().Return(func(ctx context.Context, id models.AccountID) (*models.Account, error) {
		return &acc, nil
	})
	// Expect storing the mapped balance
	s.env.OnActivity(activities.StorageBalancesStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, balances []models.Balance) error {
		s.Equal(1, len(balances))
		s.Equal(acc.ID, balances[0].AccountID)
		return nil
	})
	// SendEvents child workflow should be invoked
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunStoreWebhookTranslation, StoreWebhookTranslation{
		ConnectorID: s.connectorID,
		Balance:     &s.pspBalance,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_StoreWebhookTranslation_Balance_NoAccountFound() {
	// StorageAccountsGet returns an error; workflow should still store balance using FromPSPBalance
	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, models.AccountID{
		Reference:   s.pspBalance.AccountReference,
		ConnectorID: s.connectorID,
	}).Once().Return(
		temporal.NewNonRetryableApplicationError("not found", "not found", errors.New("not found")),
	)
	// Expect storing the balance without account context
	s.env.OnActivity(activities.StorageBalancesStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, balances []models.Balance) error {
		s.Equal(1, len(balances))
		s.Equal(models.AccountID{Reference: s.pspBalance.AccountReference, ConnectorID: s.connectorID}, balances[0].AccountID)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunStoreWebhookTranslation, StoreWebhookTranslation{
		ConnectorID: s.connectorID,
		Balance:     &s.pspBalance,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_StoreWebhookTranslation_Balance_MapperFails() {
	// Return a valid account, but craft an invalid PSPBalance (zero CreatedAt)
	acc := s.account
	acc.ID = s.accountID
	acc.Reference = s.accountID.Reference

	invalid := s.pspBalance
	invalid.CreatedAt = time.Time{}

	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, models.AccountID{
		Reference:   invalid.AccountReference,
		ConnectorID: s.connectorID,
	}).Once().Return(func(ctx context.Context, id models.AccountID) (*models.Account, error) {
		return &acc, nil
	})

	s.env.ExecuteWorkflow(RunStoreWebhookTranslation, StoreWebhookTranslation{
		ConnectorID: s.connectorID,
		Balance:     &invalid,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	// Should be translated to a non-retryable application error with this message
	s.ErrorContains(err, "failed to translate balances")
}

func (s *UnitTestSuite) Test_StoreWebhookTranslation_Balance_StorageFails() {
	// Return a valid account
	acc := s.account
	acc.ID = s.accountID
	acc.Reference = s.accountID.Reference

	s.env.OnActivity(activities.StorageAccountsGetActivity, mock.Anything, models.AccountID{
		Reference:   s.pspBalance.AccountReference,
		ConnectorID: s.connectorID,
	}).Once().Return(func(ctx context.Context, id models.AccountID) (*models.Account, error) {
		return &acc, nil
	})
	// Fail storing balances
	s.env.OnActivity(activities.StorageBalancesStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("store error", "store error", errors.New("store error")),
	)

	s.env.ExecuteWorkflow(RunStoreWebhookTranslation, StoreWebhookTranslation{
		ConnectorID: s.connectorID,
		Balance:     &s.pspBalance,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "storing next balances")
}
