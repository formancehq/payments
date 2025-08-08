package workflow

import (
	"errors"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_DeletePSUConnector_Success() {
	taskID := models.TaskID{
		Reference:   "delete-psu-connector-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
		ContactDetails: &models.ContactDetails{
			Email:       pointer.For("test@example.com"),
			PhoneNumber: pointer.For("+1234567890"),
		},
		Address: &models.Address{
			StreetName:   pointer.For("Test Street"),
			StreetNumber: pointer.For("123"),
			City:         pointer.For("Test City"),
			Region:       pointer.For("Test Region"),
			PostalCode:   pointer.For("12345"),
			Country:      pointer.For("US"),
		},
		Metadata: map[string]string{
			"source": "test",
		},
	}

	psuBankBridge := &models.PSUBankBridge{
		ConnectorID: s.connectorID,
		AccessToken: &models.Token{
			Token: "auth-token",
		},
		Metadata: map[string]string{
			"bank": "test-bank",
		},
	}

	// Mock PSU retrieval
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)

	// Mock PSU bank bridge retrieval
	s.env.OnActivity(activities.StoragePSUBankBridgesGetActivity, mock.Anything, psuID, s.connectorID).Once().Return(psuBankBridge, nil)

	// Mock plugin delete user
	s.env.OnActivity(activities.PluginDeleteUserActivity, mock.Anything, mock.Anything).Once().Return(&models.DeleteUserResponse{}, nil)

	// Mock child workflow execution
	s.env.OnWorkflow(RunDeleteBankBridgeConnectionData, mock.Anything, mock.Anything).Return(nil)

	// Mock PSU bank bridge deletion
	s.env.OnActivity(activities.StoragePSUBankBridgesDeleteActivity, mock.Anything, psuID, s.connectorID).Once().Return(nil)

	// Mock task success update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSUConnector, DeletePSUConnector{
		TaskID:      taskID,
		PsuID:       psuID,
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_DeletePSUConnector_StoragePaymentServiceUsersGet_Error() {
	taskID := models.TaskID{
		Reference:   "delete-psu-connector-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	// Mock PSU retrieval error
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(
		nil, temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	// Mock task error update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSUConnector, DeletePSUConnector{
		TaskID:      taskID,
		PsuID:       psuID,
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_DeletePSUConnector_StoragePSUBankBridgesGet_Error() {
	taskID := models.TaskID{
		Reference:   "delete-psu-connector-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
	}

	// Mock PSU retrieval
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)

	// Mock PSU bank bridge retrieval error
	s.env.OnActivity(activities.StoragePSUBankBridgesGetActivity, mock.Anything, psuID, s.connectorID).Once().Return(
		nil, temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	// Mock task error update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSUConnector, DeletePSUConnector{
		TaskID:      taskID,
		PsuID:       psuID,
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_DeletePSUConnector_PluginDeleteUser_Error() {
	taskID := models.TaskID{
		Reference:   "delete-psu-connector-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
	}

	psuBankBridge := &models.PSUBankBridge{
		ConnectorID: s.connectorID,
		AccessToken: &models.Token{
			Token: "auth-token",
		},
		Metadata: map[string]string{
			"bank": "test-bank",
		},
	}

	// Mock PSU retrieval
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)

	// Mock PSU bank bridge retrieval
	s.env.OnActivity(activities.StoragePSUBankBridgesGetActivity, mock.Anything, psuID, s.connectorID).Once().Return(psuBankBridge, nil)

	// Mock plugin delete user error
	s.env.OnActivity(activities.PluginDeleteUserActivity, mock.Anything, mock.Anything).Once().Return(
		nil, temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	// Mock task error update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSUConnector, DeletePSUConnector{
		TaskID:      taskID,
		PsuID:       psuID,
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_DeletePSUConnector_ChildWorkflow_Error() {
	taskID := models.TaskID{
		Reference:   "delete-psu-connector-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
	}

	psuBankBridge := &models.PSUBankBridge{
		ConnectorID: s.connectorID,
		AccessToken: &models.Token{
			Token: "auth-token",
		},
		Metadata: map[string]string{
			"bank": "test-bank",
		},
	}

	// Mock PSU retrieval
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)

	// Mock PSU bank bridge retrieval
	s.env.OnActivity(activities.StoragePSUBankBridgesGetActivity, mock.Anything, psuID, s.connectorID).Once().Return(psuBankBridge, nil)

	// Mock plugin delete user
	s.env.OnActivity(activities.PluginDeleteUserActivity, mock.Anything, mock.Anything).Once().Return(&models.DeleteUserResponse{}, nil)

	// Mock child workflow execution error
	s.env.OnWorkflow(RunDeleteBankBridgeConnectionData, mock.Anything, mock.Anything).Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	// Mock task error update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSUConnector, DeletePSUConnector{
		TaskID:      taskID,
		PsuID:       psuID,
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_DeletePSUConnector_StoragePSUBankBridgesDelete_Error() {
	taskID := models.TaskID{
		Reference:   "delete-psu-connector-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
	}

	psuBankBridge := &models.PSUBankBridge{
		ConnectorID: s.connectorID,
		AccessToken: &models.Token{
			Token: "auth-token",
		},
		Metadata: map[string]string{
			"bank": "test-bank",
		},
	}

	// Mock PSU retrieval
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)

	// Mock PSU bank bridge retrieval
	s.env.OnActivity(activities.StoragePSUBankBridgesGetActivity, mock.Anything, psuID, s.connectorID).Once().Return(psuBankBridge, nil)

	// Mock plugin delete user
	s.env.OnActivity(activities.PluginDeleteUserActivity, mock.Anything, mock.Anything).Once().Return(&models.DeleteUserResponse{}, nil)

	// Mock child workflow execution
	s.env.OnWorkflow(RunDeleteBankBridgeConnectionData, mock.Anything, mock.Anything).Return(nil)

	// Mock PSU bank bridge deletion error
	s.env.OnActivity(activities.StoragePSUBankBridgesDeleteActivity, mock.Anything, psuID, s.connectorID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	// Mock task error update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSUConnector, DeletePSUConnector{
		TaskID:      taskID,
		PsuID:       psuID,
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_DeletePSUConnector_TaskErrorUpdate_Error() {
	taskID := models.TaskID{
		Reference:   "delete-psu-connector-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	// Mock PSU retrieval error
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(
		nil, temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	// Mock task error update error
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeletePSUConnector, DeletePSUConnector{
		TaskID:      taskID,
		PsuID:       psuID,
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_DeletePSUConnector_TaskSuccessUpdate_Error() {
	taskID := models.TaskID{
		Reference:   "delete-psu-connector-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
	}

	psuBankBridge := &models.PSUBankBridge{
		ConnectorID: s.connectorID,
		AccessToken: &models.Token{
			Token: "auth-token",
		},
		Metadata: map[string]string{
			"bank": "test-bank",
		},
	}

	// Mock PSU retrieval
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)

	// Mock PSU bank bridge retrieval
	s.env.OnActivity(activities.StoragePSUBankBridgesGetActivity, mock.Anything, psuID, s.connectorID).Once().Return(psuBankBridge, nil)

	// Mock plugin delete user
	s.env.OnActivity(activities.PluginDeleteUserActivity, mock.Anything, mock.Anything).Once().Return(&models.DeleteUserResponse{}, nil)

	// Mock child workflow execution
	s.env.OnWorkflow(RunDeleteBankBridgeConnectionData, mock.Anything, mock.Anything).Return(nil)

	// Mock PSU bank bridge deletion
	s.env.OnActivity(activities.StoragePSUBankBridgesDeleteActivity, mock.Anything, psuID, s.connectorID).Once().Return(nil)

	// Mock task success update error
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunDeletePSUConnector, DeletePSUConnector{
		TaskID:      taskID,
		PsuID:       psuID,
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_DeletePSUConnector_WithMinimalPSU() {
	taskID := models.TaskID{
		Reference:   "delete-psu-connector-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		CreatedAt: s.env.Now().UTC(),
	}

	psuBankBridge := &models.PSUBankBridge{
		ConnectorID: s.connectorID,
		AccessToken: &models.Token{
			Token: "auth-token",
		},
	}

	// Mock PSU retrieval
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)

	// Mock PSU bank bridge retrieval
	s.env.OnActivity(activities.StoragePSUBankBridgesGetActivity, mock.Anything, psuID, s.connectorID).Once().Return(psuBankBridge, nil)

	// Mock plugin delete user
	s.env.OnActivity(activities.PluginDeleteUserActivity, mock.Anything, mock.Anything).Once().Return(&models.DeleteUserResponse{}, nil)

	// Mock child workflow execution
	s.env.OnWorkflow(RunDeleteBankBridgeConnectionData, mock.Anything, mock.Anything).Return(nil)

	// Mock PSU bank bridge deletion
	s.env.OnActivity(activities.StoragePSUBankBridgesDeleteActivity, mock.Anything, psuID, s.connectorID).Once().Return(nil)

	// Mock task success update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSUConnector, DeletePSUConnector{
		TaskID:      taskID,
		PsuID:       psuID,
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_DeletePSUConnector_WithPSUWithoutBankBridge() {
	taskID := models.TaskID{
		Reference:   "delete-psu-connector-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
	}

	// Mock PSU retrieval
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)

	// Mock PSU bank bridge retrieval - no bank bridge found
	s.env.OnActivity(activities.StoragePSUBankBridgesGetActivity, mock.Anything, psuID, s.connectorID).Once().Return(
		nil, temporal.NewNonRetryableApplicationError("not found", "not found", errors.New("not found")),
	)

	// Mock task error update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSUConnector, DeletePSUConnector{
		TaskID:      taskID,
		PsuID:       psuID,
		ConnectorID: s.connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "not found")
}
