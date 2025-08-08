package workflow

import (
	"errors"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_DeletePSU_Success() {
	taskID := models.TaskID{
		Reference:   "delete-psu-task",
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

	psuBankBridges := &bunpaginate.Cursor[models.PSUBankBridge]{
		Data: []models.PSUBankBridge{
			{
				ConnectorID: s.connectorID,
				AccessToken: &models.Token{
					Token: "auth-token-1",
				},
				Metadata: map[string]string{
					"bank": "test-bank-1",
				},
			},
			{
				ConnectorID: s.connectorID,
				AccessToken: &models.Token{
					Token: "auth-token-2",
				},
				Metadata: map[string]string{
					"bank": "test-bank-2",
				},
			},
		},
		HasMore: false,
	}

	// Mock PSU retrieval
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)

	// Mock PSU bank bridges list
	s.env.OnActivity(activities.StoragePSUBankBridgesListActivity, mock.Anything, mock.Anything).Once().Return(psuBankBridges, nil)

	// Mock plugin delete user (called for each bank bridge)
	s.env.OnActivity(activities.PluginDeleteUserActivity, mock.Anything, mock.Anything, mock.Anything).Return(&models.DeleteUserResponse{}, nil)
	s.env.OnActivity(activities.PluginDeleteUserActivity, mock.Anything, mock.Anything, mock.Anything).Return(&models.DeleteUserResponse{}, nil)

	// Mock child workflow execution
	s.env.OnWorkflow(RunDeleteBankBridgeConnectionData, mock.Anything, mock.Anything).Return(nil)

	// Mock PSU deletion
	s.env.OnActivity(activities.StoragePaymentServiceUsersDeleteActivity, mock.Anything, psuID.String()).Once().Return(nil)

	// Mock task success update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSU, DeletePSU{
		TaskID: taskID,
		PsuID:  psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_DeletePSU_StoragePaymentServiceUsersGet_Error() {
	taskID := models.TaskID{
		Reference:   "delete-psu-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	// Mock PSU retrieval error
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(
		nil, temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("user not found")),
	)

	// Mock task error update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSU, DeletePSU{
		TaskID: taskID,
		PsuID:  psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "user not found")
}

func (s *UnitTestSuite) Test_DeletePSU_StoragePSUBankBridgesList_Error() {
	taskID := models.TaskID{
		Reference:   "delete-psu-task",
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

	// Mock PSU bank bridges list error
	s.env.OnActivity(activities.StoragePSUBankBridgesListActivity, mock.Anything, mock.Anything).Once().Return(
		nil, temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("database error")),
	)

	// Mock task error update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSU, DeletePSU{
		TaskID: taskID,
		PsuID:  psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "database error")
}

func (s *UnitTestSuite) Test_DeletePSU_PluginDeleteUser_Error() {
	taskID := models.TaskID{
		Reference:   "delete-psu-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
	}

	psuBankBridges := &bunpaginate.Cursor[models.PSUBankBridge]{
		Data: []models.PSUBankBridge{
			{
				ConnectorID: s.connectorID,
				AccessToken: &models.Token{
					Token: "auth-token-1",
				},
				Metadata: map[string]string{
					"bank": "test-bank-1",
				},
			},
		},
		HasMore: false,
	}

	// Mock PSU retrieval
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)

	// Mock PSU bank bridges list
	s.env.OnActivity(activities.StoragePSUBankBridgesListActivity, mock.Anything, mock.Anything).Once().Return(psuBankBridges, nil)

	// Mock plugin delete user error
	s.env.OnActivity(activities.PluginDeleteUserActivity, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		nil, temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("plugin error")),
	)

	// Mock task error update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSU, DeletePSU{
		TaskID: taskID,
		PsuID:  psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "plugin error")
}

func (s *UnitTestSuite) Test_DeletePSU_ChildWorkflow_Error() {
	taskID := models.TaskID{
		Reference:   "delete-psu-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
	}

	psuBankBridges := &bunpaginate.Cursor[models.PSUBankBridge]{
		Data: []models.PSUBankBridge{
			{
				ConnectorID: s.connectorID,
				AccessToken: &models.Token{
					Token: "auth-token-1",
				},
				Metadata: map[string]string{
					"bank": "test-bank-1",
				},
			},
		},
		HasMore: false,
	}

	// Mock PSU retrieval
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)

	// Mock PSU bank bridges list
	s.env.OnActivity(activities.StoragePSUBankBridgesListActivity, mock.Anything, mock.Anything).Once().Return(psuBankBridges, nil)

	// Mock plugin delete user
	s.env.OnActivity(activities.PluginDeleteUserActivity, mock.Anything, mock.Anything, mock.Anything).Once().Return(&models.DeleteUserResponse{}, nil)

	// Mock child workflow error
	s.env.OnWorkflow(RunDeleteBankBridgeConnectionData, mock.Anything, mock.Anything).Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("child workflow error")),
	)

	// Mock task error update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSU, DeletePSU{
		TaskID: taskID,
		PsuID:  psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "child workflow error")
}

func (s *UnitTestSuite) Test_DeletePSU_StoragePaymentServiceUsersDelete_Error() {
	taskID := models.TaskID{
		Reference:   "delete-psu-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
	}

	psuBankBridges := &bunpaginate.Cursor[models.PSUBankBridge]{
		Data: []models.PSUBankBridge{
			{
				ConnectorID: s.connectorID,
				AccessToken: &models.Token{
					Token: "auth-token-1",
				},
				Metadata: map[string]string{
					"bank": "test-bank-1",
				},
			},
		},
		HasMore: false,
	}

	// Mock PSU retrieval
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)

	// Mock PSU bank bridges list
	s.env.OnActivity(activities.StoragePSUBankBridgesListActivity, mock.Anything, mock.Anything).Once().Return(psuBankBridges, nil)

	// Mock plugin delete user
	s.env.OnActivity(activities.PluginDeleteUserActivity, mock.Anything, mock.Anything, mock.Anything).Once().Return(&models.DeleteUserResponse{}, nil)

	// Mock child workflow execution
	s.env.OnWorkflow(RunDeleteBankBridgeConnectionData, mock.Anything, mock.Anything).Return(nil)

	// Mock PSU deletion error
	s.env.OnActivity(activities.StoragePaymentServiceUsersDeleteActivity, mock.Anything, psuID.String()).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("deletion error")),
	)

	// Mock task error update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSU, DeletePSU{
		TaskID: taskID,
		PsuID:  psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "deletion error")
}

func (s *UnitTestSuite) Test_DeletePSU_UpdateTaskError_Error() {
	taskID := models.TaskID{
		Reference:   "delete-psu-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	// Mock PSU retrieval error
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(
		nil, temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("user not found")),
	)

	// Mock task error update error
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("task update error")),
	)

	s.env.ExecuteWorkflow(RunDeletePSU, DeletePSU{
		TaskID: taskID,
		PsuID:  psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_DeletePSU_UpdateTaskSuccess_Error() {
	taskID := models.TaskID{
		Reference:   "delete-psu-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
	}

	psuBankBridges := &bunpaginate.Cursor[models.PSUBankBridge]{
		Data: []models.PSUBankBridge{
			{
				ConnectorID: s.connectorID,
				AccessToken: &models.Token{
					Token: "auth-token-1",
				},
				Metadata: map[string]string{
					"bank": "test-bank-1",
				},
			},
		},
		HasMore: false,
	}

	// Mock PSU retrieval
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)

	// Mock PSU bank bridges list
	s.env.OnActivity(activities.StoragePSUBankBridgesListActivity, mock.Anything, mock.Anything).Once().Return(psuBankBridges, nil)

	// Mock plugin delete user
	s.env.OnActivity(activities.PluginDeleteUserActivity, mock.Anything, mock.Anything, mock.Anything).Once().Return(&models.DeleteUserResponse{}, nil)

	// Mock child workflow execution
	s.env.OnWorkflow(RunDeleteBankBridgeConnectionData, mock.Anything, mock.Anything).Return(nil)

	// Mock PSU deletion
	s.env.OnActivity(activities.StoragePaymentServiceUsersDeleteActivity, mock.Anything, psuID.String()).Once().Return(nil)

	// Mock task success update error
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("task update error")),
	)

	s.env.ExecuteWorkflow(RunDeletePSU, DeletePSU{
		TaskID: taskID,
		PsuID:  psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_DeletePSU_WithMultipleBankBridges() {
	taskID := models.TaskID{
		Reference:   "delete-psu-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
	}

	// Multiple bank bridges in a single page
	psuBankBridges := &bunpaginate.Cursor[models.PSUBankBridge]{
		Data: []models.PSUBankBridge{
			{
				ConnectorID: s.connectorID,
				AccessToken: &models.Token{
					Token: "auth-token-1",
				},
				Metadata: map[string]string{
					"bank": "test-bank-1",
				},
			},
			{
				ConnectorID: s.connectorID,
				AccessToken: &models.Token{
					Token: "auth-token-2",
				},
				Metadata: map[string]string{
					"bank": "test-bank-2",
				},
			},
			{
				ConnectorID: s.connectorID,
				AccessToken: &models.Token{
					Token: "auth-token-3",
				},
				Metadata: map[string]string{
					"bank": "test-bank-3",
				},
			},
		},
		HasMore: false,
	}

	// Mock PSU retrieval
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)

	// Mock PSU bank bridges list
	s.env.OnActivity(activities.StoragePSUBankBridgesListActivity, mock.Anything, mock.Anything).Once().Return(psuBankBridges, nil)

	// Mock plugin delete user (called for each bank bridge - 3 total)
	s.env.OnActivity(activities.PluginDeleteUserActivity, mock.Anything, mock.Anything, mock.Anything).Return(&models.DeleteUserResponse{}, nil)
	s.env.OnActivity(activities.PluginDeleteUserActivity, mock.Anything, mock.Anything, mock.Anything).Return(&models.DeleteUserResponse{}, nil)
	s.env.OnActivity(activities.PluginDeleteUserActivity, mock.Anything, mock.Anything, mock.Anything).Return(&models.DeleteUserResponse{}, nil)

	// Mock child workflow execution
	s.env.OnWorkflow(RunDeleteBankBridgeConnectionData, mock.Anything, mock.Anything).Return(nil)

	// Mock PSU deletion
	s.env.OnActivity(activities.StoragePaymentServiceUsersDeleteActivity, mock.Anything, psuID.String()).Once().Return(nil)

	// Mock task success update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSU, DeletePSU{
		TaskID: taskID,
		PsuID:  psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_DeletePSU_WithNoBankBridges() {
	taskID := models.TaskID{
		Reference:   "delete-psu-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
	}

	// Empty bank bridges list
	psuBankBridges := &bunpaginate.Cursor[models.PSUBankBridge]{
		Data:    []models.PSUBankBridge{},
		HasMore: false,
	}

	// Mock PSU retrieval
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)

	// Mock PSU bank bridges list
	s.env.OnActivity(activities.StoragePSUBankBridgesListActivity, mock.Anything, mock.Anything).Once().Return(psuBankBridges, nil)

	// Mock child workflow execution
	s.env.OnWorkflow(RunDeleteBankBridgeConnectionData, mock.Anything, mock.Anything).Return(nil)

	// Mock PSU deletion
	s.env.OnActivity(activities.StoragePaymentServiceUsersDeleteActivity, mock.Anything, psuID.String()).Once().Return(nil)

	// Mock task success update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSU, DeletePSU{
		TaskID: taskID,
		PsuID:  psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_DeletePSU_WithMinimalPSU() {
	taskID := models.TaskID{
		Reference:   "delete-psu-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()

	// Minimal PSU with only required fields
	psu := &models.PaymentServiceUser{
		ID:        psuID,
		CreatedAt: s.env.Now().UTC(),
	}

	psuBankBridges := &bunpaginate.Cursor[models.PSUBankBridge]{
		Data: []models.PSUBankBridge{
			{
				ConnectorID: s.connectorID,
				AccessToken: &models.Token{
					Token: "auth-token-1",
				},
			},
		},
		HasMore: false,
	}

	// Mock PSU retrieval
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)

	// Mock PSU bank bridges list
	s.env.OnActivity(activities.StoragePSUBankBridgesListActivity, mock.Anything, mock.Anything).Once().Return(psuBankBridges, nil)

	// Mock plugin delete user
	s.env.OnActivity(activities.PluginDeleteUserActivity, mock.Anything, mock.Anything, mock.Anything).Once().Return(&models.DeleteUserResponse{}, nil)

	// Mock child workflow execution
	s.env.OnWorkflow(RunDeleteBankBridgeConnectionData, mock.Anything, mock.Anything).Return(nil)

	// Mock PSU deletion
	s.env.OnActivity(activities.StoragePaymentServiceUsersDeleteActivity, mock.Anything, psuID.String()).Once().Return(nil)

	// Mock task success update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSU, DeletePSU{
		TaskID: taskID,
		PsuID:  psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}
