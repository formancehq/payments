package workflow

import (
	"context"
	"errors"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_UpdatePaymentInitiationFromPayment_Success() {
	//nolint:staticcheck // ignore deprecation
	s.env.OnActivity(activities.StoragePaymentInitiationIDsListFromPaymentIDActivity, mock.Anything, s.paymentPayoutID).Once().Return(
		[]models.PaymentInitiationID{s.paymentInitiationID},
		nil,
	)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, adj models.PaymentInitiationAdjustment) error {
		s.Equal(s.paymentInitiationID, adj.ID.PaymentInitiationID)
		return nil
	})

	//nolint:staticcheck // ignore deprecation
	s.env.ExecuteWorkflow(RunUpdatePaymentInitiationFromPayment, UpdatePaymentInitiationFromPayment{
		Payment: &s.paymentPayout,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_UpdatePaymentInitiationFromPayment_NoIds_Success() {
	//nolint:staticcheck // ignore deprecation
	s.env.OnActivity(activities.StoragePaymentInitiationIDsListFromPaymentIDActivity, mock.Anything, s.paymentPayoutID).Once().Return(
		[]models.PaymentInitiationID{},
		nil,
	)

	//nolint:staticcheck // ignore deprecation
	s.env.ExecuteWorkflow(RunUpdatePaymentInitiationFromPayment, UpdatePaymentInitiationFromPayment{
		Payment: &s.paymentPayout,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_UpdatePaymentInitiationFromPayment_SkipAdjustment_Success() {
	//nolint:staticcheck // ignore deprecation
	s.env.OnActivity(activities.StoragePaymentInitiationIDsListFromPaymentIDActivity, mock.Anything, s.paymentPayoutID).Once().Return(
		[]models.PaymentInitiationID{s.paymentInitiationID},
		nil,
	)
	//nolint:staticcheck // ignore deprecation
	s.env.ExecuteWorkflow(RunUpdatePaymentInitiationFromPayment, UpdatePaymentInitiationFromPayment{
		Payment: &s.paymentWithAdjustmentAmount,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_UpdatePaymentInitiationFromPayment_StoragePaymentInitiationIDsListFromPaymentID_Error() {
	//nolint:staticcheck // ignore deprecation
	s.env.OnActivity(activities.StoragePaymentInitiationIDsListFromPaymentIDActivity, mock.Anything, s.paymentPayoutID).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	//nolint:staticcheck // ignore deprecation
	s.env.ExecuteWorkflow(RunUpdatePaymentInitiationFromPayment, UpdatePaymentInitiationFromPayment{
		Payment: &s.paymentPayout,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_UpdatePaymentInitiationFromPayment_StoragePaymentInitiationsAdjustmentsStore_Error() {
	//nolint:staticcheck // ignore deprecation
	s.env.OnActivity(activities.StoragePaymentInitiationIDsListFromPaymentIDActivity, mock.Anything, s.paymentPayoutID).Once().Return(
		[]models.PaymentInitiationID{s.paymentInitiationID},
		nil,
	)
	s.env.OnActivity(activities.StoragePaymentInitiationsAdjustmentsStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	//nolint:staticcheck // ignore deprecation
	s.env.ExecuteWorkflow(RunUpdatePaymentInitiationFromPayment, UpdatePaymentInitiationFromPayment{
		Payment: &s.paymentPayout,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}
