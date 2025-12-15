package workflow

import (
	"context"
	"errors"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_FetchNextPayments_WithoutInstance_Success() {
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_PAYMENTS.String(), "1"),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextPaymentsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextPaymentsRequest) (*models.FetchNextPaymentsResponse, error) {
		return &models.FetchNextPaymentsResponse{
			Payments: []models.PSPPayment{
				s.pspPayment,
			},
			NewState: []byte(`{}`),
			HasMore:  false,
		}, nil
	})
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, payments []models.Payment) error {
		s.Equal(1, len(payments))
		s.Equal(s.paymentPayoutID, payments[0].ID)
		return nil
	})
	s.env.OnActivity(activities.StoragePaymentInitiationUpdateFromPaymentActivity, mock.Anything, s.pspPayment.Status, s.pspPayment.CreatedAt, s.paymentPayoutID).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchNextPayments, FetchNextPayments{
		Config:      models.Config{},
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

func (s *UnitTestSuite) Test_FetchNextPayments_WithNextTasks_Success() {
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_PAYMENTS.String(), "1"),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextPaymentsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextPaymentsRequest) (*models.FetchNextPaymentsResponse, error) {
		return &models.FetchNextPaymentsResponse{
			Payments: []models.PSPPayment{
				s.pspPayment,
			},
			NewState: []byte(`{}`),
			HasMore:  false,
		}, nil
	})
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, payments []models.Payment) error {
		s.Equal(1, len(payments))
		s.Equal(s.paymentPayoutID, payments[0].ID)
		return nil
	})
	s.env.OnActivity(activities.StoragePaymentInitiationUpdateFromPaymentActivity, mock.Anything, s.pspPayment.Status, s.pspPayment.CreatedAt, s.paymentPayoutID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorsGetMetadataActivity, mock.Anything, s.connectorID).Once().Return(
		&models.ConnectorMetadata{
			ConnectorID:          s.connector.ID,
			Provider:             s.connector.Provider,
			PollingPeriod:        models.DefaultConfig().PollingPeriod,
			ScheduledForDeletion: s.connector.ScheduledForDeletion,
		},
		nil,
	)
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleCreateActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchNextPayments, FetchNextPayments{
		Config:      models.Config{},
		ConnectorID: s.connectorID,
		FromPayload: &FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		Periodically: false,
	}, []models.ConnectorTaskTree{
		{
			TaskType:     models.TASK_FETCH_PAYMENTS,
			Name:         "test",
			Periodically: true,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextPayments_WithNextTasks_ConnectorScheduledForDeletion_Success() {
	connector := s.connector
	connector.ScheduledForDeletion = true
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_PAYMENTS.String(), "1"),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextPaymentsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextPaymentsRequest) (*models.FetchNextPaymentsResponse, error) {
		return &models.FetchNextPaymentsResponse{
			Payments: []models.PSPPayment{
				s.pspPayment,
			},
			NewState: []byte(`{}`),
			HasMore:  false,
		}, nil
	})
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, payments []models.Payment) error {
		s.Equal(1, len(payments))
		s.Equal(s.paymentPayoutID, payments[0].ID)
		return nil
	})
	s.env.OnActivity(activities.StoragePaymentInitiationUpdateFromPaymentActivity, mock.Anything, s.pspPayment.Status, s.pspPayment.CreatedAt, s.paymentPayoutID).Once().Return(nil)
	s.env.OnActivity(activities.StorageConnectorsGetMetadataActivity, mock.Anything, s.connectorID).Once().Return(
		&models.ConnectorMetadata{
			ConnectorID:          connector.ID,
			Provider:             connector.Provider,
			PollingPeriod:        models.DefaultConfig().PollingPeriod,
			ScheduledForDeletion: connector.ScheduledForDeletion,
		},
		nil,
	)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchNextPayments, FetchNextPayments{
		Config:      models.Config{},
		ConnectorID: s.connectorID,
		FromPayload: &FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		Periodically: false,
	}, []models.ConnectorTaskTree{
		{
			TaskType:     models.TASK_FETCH_PAYMENTS,
			Name:         "test",
			Periodically: true,
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextPayments_Success() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.Equal("test", instance.ScheduleID)
		s.Equal(s.connectorID, instance.ConnectorID)
		s.False(instance.Terminated)
		return nil
	})
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_PAYMENTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextPaymentsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextPaymentsRequest) (*models.FetchNextPaymentsResponse, error) {
		return &models.FetchNextPaymentsResponse{
			Payments: []models.PSPPayment{
				s.pspPayment,
			},
			NewState: []byte(`{}`),
			HasMore:  false,
		}, nil
	})
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, payments []models.Payment) error {
		s.Equal(1, len(payments))
		s.Equal(s.paymentPayoutID, payments[0].ID)
		return nil
	})

	s.env.OnActivity(activities.StoragePaymentInitiationUpdateFromPaymentActivity, mock.Anything, s.pspPayment.Status, s.pspPayment.CreatedAt, s.paymentPayoutID).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.Equal("test", instance.ScheduleID)
		s.Equal(s.connectorID, instance.ConnectorID)
		s.True(instance.Terminated)
		return nil
	})

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.NoError(err)
	s.env.ExecuteWorkflow(RunFetchNextPayments, FetchNextPayments{
		Config:       models.Config{},
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextPayments_WithoutNextTasks_Success() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.Equal("test", instance.ScheduleID)
		s.Equal(s.connectorID, instance.ConnectorID)
		s.False(instance.Terminated)
		return nil
	})
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_PAYMENTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextPaymentsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextPaymentsRequest) (*models.FetchNextPaymentsResponse, error) {
		return &models.FetchNextPaymentsResponse{
			Payments: []models.PSPPayment{
				s.pspPayment,
			},
			NewState: []byte(`{}`),
			HasMore:  false,
		}, nil
	})
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, payments []models.Payment) error {
		s.Equal(1, len(payments))
		s.Equal(s.paymentPayoutID, payments[0].ID)
		return nil
	})
	s.env.OnActivity(activities.StoragePaymentInitiationUpdateFromPaymentActivity, mock.Anything, s.pspPayment.Status, s.pspPayment.CreatedAt, s.paymentPayoutID).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.Equal("test", instance.ScheduleID)
		s.Equal(s.connectorID, instance.ConnectorID)
		s.True(instance.Terminated)
		return nil
	})

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.NoError(err)
	s.env.ExecuteWorkflow(RunFetchNextPayments, FetchNextPayments{
		Config:       models.Config{},
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextPayments_HasMoreLoop_Success() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.Equal("test", instance.ScheduleID)
		s.Equal(s.connectorID, instance.ConnectorID)
		s.False(instance.Terminated)
		return nil
	})
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_PAYMENTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextPaymentsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextPaymentsRequest) (*models.FetchNextPaymentsResponse, error) {
		return &models.FetchNextPaymentsResponse{
			Payments: []models.PSPPayment{
				s.pspPayment,
			},
			NewState: []byte(`{}`),
			HasMore:  true,
		}, nil
	})
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, payments []models.Payment) error {
		s.Equal(1, len(payments))
		s.Equal(s.paymentPayoutID, payments[0].ID)
		return nil
	})
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.OnActivity(activities.PluginFetchNextPaymentsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextPaymentsRequest) (*models.FetchNextPaymentsResponse, error) {
		return &models.FetchNextPaymentsResponse{
			Payments: []models.PSPPayment{},
			NewState: []byte(`{}`),
			HasMore:  false,
		}, nil
	})
	s.env.OnActivity(activities.StoragePaymentInitiationUpdateFromPaymentActivity, mock.Anything, s.pspPayment.Status, s.pspPayment.CreatedAt, s.paymentPayoutID).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.Equal("test", instance.ScheduleID)
		s.Equal(s.connectorID, instance.ConnectorID)
		s.True(instance.Terminated)
		return nil
	})

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.NoError(err)
	s.env.ExecuteWorkflow(RunFetchNextPayments, FetchNextPayments{
		Config:       models.Config{},
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextPayments_StorageInstancesStore_Error() {
	expectedErr := errors.New("error-test")
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "STORAGE", expectedErr),
	)

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.NoError(err)
	s.env.ExecuteWorkflow(RunFetchNextPayments, FetchNextPayments{
		Config:       models.Config{},
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, expectedErr.Error())
}

func (s *UnitTestSuite) Test_FetchNextPayments_StorageStatesGet_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	expectedErr := errors.New("error-test")
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("error-test", "STORAGE", expectedErr),
	)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.True(instance.Terminated)
		s.NotNil(instance.Error)
		return nil
	})

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.NoError(err)
	s.env.ExecuteWorkflow(RunFetchNextPayments, FetchNextPayments{
		Config:       models.Config{},
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, expectedErr.Error())
}

func (s *UnitTestSuite) Test_FetchNextPayments_PluginFetchNextPayments_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_PAYMENTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	expectedErr := errors.New("error-test")
	s.env.OnActivity(activities.PluginFetchNextPaymentsActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("error-test", "PLUGIN", expectedErr),
	)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.True(instance.Terminated)
		s.NotNil(instance.Error)
		return nil
	})

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.NoError(err)
	s.env.ExecuteWorkflow(RunFetchNextPayments, FetchNextPayments{
		Config:       models.Config{},
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, expectedErr.Error())
}

func (s *UnitTestSuite) Test_FetchNextPayments_StoragePaymentsStore_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_PAYMENTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextPaymentsActivity, mock.Anything, mock.Anything).Once().Return(&models.FetchNextPaymentsResponse{
		Payments: []models.PSPPayment{
			s.pspPayment,
		},
		NewState: []byte(`{}`),
		HasMore:  false,
	}, nil)
	expectedErr := errors.New("error-test")
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "STORAGE", expectedErr),
	)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.True(instance.Terminated)
		s.NotNil(instance.Error)
		return nil
	})

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.NoError(err)
	s.env.ExecuteWorkflow(RunFetchNextPayments, FetchNextPayments{
		Config:       models.Config{},
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, expectedErr.Error())
}

func (s *UnitTestSuite) Test_FetchNextPayments_StoragePaymentInitiationUpdateFromPaymentActivity_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_PAYMENTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextPaymentsActivity, mock.Anything, mock.Anything).Once().Return(&models.FetchNextPaymentsResponse{
		Payments: []models.PSPPayment{
			s.pspPayment,
		},
		NewState: []byte(`{}`),
		HasMore:  false,
	}, nil)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	expectedErr := errors.New("error-test")
	s.env.OnActivity(activities.StoragePaymentInitiationUpdateFromPaymentActivity, mock.Anything, s.pspPayment.Status, s.pspPayment.CreatedAt, s.paymentPayoutID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "WORKFLOW", expectedErr),
	)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.True(instance.Terminated)
		s.NotNil(instance.Error)
		return nil
	})

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.NoError(err)
	s.env.ExecuteWorkflow(RunFetchNextPayments, FetchNextPayments{
		Config:       models.Config{},
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, expectedErr.Error())
}

func (s *UnitTestSuite) Test_FetchNextPayments_StorageStatesStore_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_PAYMENTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextPaymentsActivity, mock.Anything, mock.Anything).Once().Return(&models.FetchNextPaymentsResponse{
		Payments: []models.PSPPayment{
			s.pspPayment,
		},
		NewState: []byte(`{}`),
		HasMore:  false,
	}, nil)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationUpdateFromPaymentActivity, mock.Anything, s.pspPayment.Status, s.pspPayment.CreatedAt, s.paymentPayoutID).Once().Return(nil)
	expectedErr := errors.New("error-test")
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "STORAGE", expectedErr),
	)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.True(instance.Terminated)
		s.NotNil(instance.Error)
		return nil
	})

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.NoError(err)
	s.env.ExecuteWorkflow(RunFetchNextPayments, FetchNextPayments{
		Config:       models.Config{},
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, expectedErr.Error())
}

func (s *UnitTestSuite) Test_FetchNextPayments_StorageInstancesUpdate_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_PAYMENTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextPaymentsActivity, mock.Anything, mock.Anything).Once().Return(&models.FetchNextPaymentsResponse{
		Payments: []models.PSPPayment{
			s.pspPayment,
		},
		NewState: []byte(`{}`),
		HasMore:  false,
	}, nil)
	s.env.OnActivity(activities.StoragePaymentsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StoragePaymentInitiationUpdateFromPaymentActivity, mock.Anything, s.pspPayment.Status, s.pspPayment.CreatedAt, s.paymentPayoutID).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	expectedErr := errors.New("error-test")
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.True(instance.Terminated)
		s.Nil(instance.Error)
		return temporal.NewNonRetryableApplicationError("error-test", "STORAGE", expectedErr)
	})

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.NoError(err)
	s.env.ExecuteWorkflow(RunFetchNextPayments, FetchNextPayments{
		Config:       models.Config{},
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, expectedErr.Error())
}
