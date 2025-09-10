package workflow

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_DeletePSUConnection_Success() {
	taskID := models.TaskID{
		Reference:   "test-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()
	connectionID := "test-connection-id"

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		CreatedAt: s.env.Now().UTC(),
	}

	connection := &models.OpenBankingConnection{
		ConnectionID:  connectionID,
		ConnectorID:   s.connectorID,
		CreatedAt:     s.env.Now().UTC(),
		DataUpdatedAt: s.env.Now().UTC(),
		Status:        models.ConnectionStatusActive,
		AccessToken: &models.Token{
			Token: "access-token",
		},
		Metadata: map[string]string{
			"bank": "test-bank",
		},
	}

	openBankingPSU := &models.OpenBankingForwardedUser{
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

	// Mock connection retrieval
	s.env.OnActivity(activities.StorageOpenBankingConnectionsGetFromConnectionIDActivity, mock.Anything, s.connectorID, connectionID).Once().Return(
		&activities.StorageOpenBankingConnectionsGetFromConnectionIDResult{
			Connection: connection,
			PSUID:      psuID,
		}, nil,
	)

	// Mock open banking forwarded user retrieval
	s.env.OnActivity(activities.StorageOpenBankingForwardedUsersGetActivity, mock.Anything, psuID, s.connectorID).Once().Return(openBankingPSU, nil)

	// Mock plugin delete user connection (multiple calls for retries)
	s.env.OnActivity(activities.PluginDeleteUserConnectionActivity, mock.Anything, mock.Anything).Return(&models.DeleteUserConnectionResponse{}, nil)

	// Mock child workflow execution
	s.env.OnWorkflow(RunDeleteOpenBankingConnectionData, mock.Anything, mock.Anything).Return(nil)

	// Mock connection deletion
	s.env.OnActivity(activities.StorageOpenBankingConnectionsDeleteActivity, mock.Anything, psuID, s.connectorID, connectionID).Once().Return(nil)

	// Mock task success update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSUConnection, DeletePSUConnection{
		TaskID:       taskID,
		ConnectorID:  s.connectorID,
		PsuID:        psuID,
		ConnectionID: connectionID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_DeletePSUConnection_StoragePaymentServiceUsersGet_Error() {
	taskID := models.TaskID{
		Reference:   "test-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()
	connectionID := "test-connection-id"

	// Mock PSU retrieval error
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(
		nil, temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	// Mock task error update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSUConnection, DeletePSUConnection{
		TaskID:       taskID,
		ConnectorID:  s.connectorID,
		PsuID:        psuID,
		ConnectionID: connectionID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_DeletePSUConnection_StoragePSUOpenBankingConnectionsGetFromConnectionID_Error() {
	taskID := models.TaskID{
		Reference:   "test-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()
	connectionID := "test-connection-id"

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		CreatedAt: s.env.Now().UTC(),
	}

	// Mock PSU retrieval
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)

	// Mock connection retrieval error
	s.env.OnActivity(activities.StorageOpenBankingConnectionsGetFromConnectionIDActivity, mock.Anything, s.connectorID, connectionID).Once().Return(
		nil, temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	// Mock task error update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSUConnection, DeletePSUConnection{
		TaskID:       taskID,
		ConnectorID:  s.connectorID,
		PsuID:        psuID,
		ConnectionID: connectionID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_DeletePSUConnection_StorageOpenBankingForwardedUserGet_Error() {
	taskID := models.TaskID{
		Reference:   "test-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()
	connectionID := "test-connection-id"

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		CreatedAt: s.env.Now().UTC(),
	}

	connection := &models.OpenBankingConnection{
		ConnectionID:  connectionID,
		ConnectorID:   s.connectorID,
		CreatedAt:     s.env.Now().UTC(),
		DataUpdatedAt: s.env.Now().UTC(),
		Status:        models.ConnectionStatusActive,
		AccessToken: &models.Token{
			Token: "access-token",
		},
		Metadata: map[string]string{
			"bank": "test-bank",
		},
	}

	// Mock PSU retrieval
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)

	// Mock connection retrieval
	s.env.OnActivity(activities.StorageOpenBankingConnectionsGetFromConnectionIDActivity, mock.Anything, s.connectorID, connectionID).Once().Return(
		&activities.StorageOpenBankingConnectionsGetFromConnectionIDResult{
			Connection: connection,
			PSUID:      psuID,
		}, nil,
	)

	// Mock open banking forwarded user retrieval error
	s.env.OnActivity(activities.StorageOpenBankingForwardedUsersGetActivity, mock.Anything, psuID, s.connectorID).Once().Return(
		nil, temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	// Mock task error update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSUConnection, DeletePSUConnection{
		TaskID:       taskID,
		ConnectorID:  s.connectorID,
		PsuID:        psuID,
		ConnectionID: connectionID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_DeletePSUConnection_PluginDeleteUserConnection_Error() {
	taskID := models.TaskID{
		Reference:   "test-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()
	connectionID := "test-connection-id"

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		CreatedAt: s.env.Now().UTC(),
	}

	connection := &models.OpenBankingConnection{
		ConnectionID:  connectionID,
		ConnectorID:   s.connectorID,
		CreatedAt:     s.env.Now().UTC(),
		DataUpdatedAt: s.env.Now().UTC(),
		Status:        models.ConnectionStatusActive,
		AccessToken: &models.Token{
			Token: "access-token",
		},
		Metadata: map[string]string{
			"bank": "test-bank",
		},
	}

	openBankingForwardedUser := &models.OpenBankingForwardedUser{
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

	// Mock connection retrieval
	s.env.OnActivity(activities.StorageOpenBankingConnectionsGetFromConnectionIDActivity, mock.Anything, s.connectorID, connectionID).Once().Return(
		&activities.StorageOpenBankingConnectionsGetFromConnectionIDResult{
			Connection: connection,
			PSUID:      psuID,
		}, nil,
	)

	// Mock open banking forwarded user retrieval
	s.env.OnActivity(activities.StorageOpenBankingForwardedUsersGetActivity, mock.Anything, psuID, s.connectorID).Once().Return(openBankingForwardedUser, nil)

	// Mock plugin delete user connection error (multiple calls for retries)
	s.env.OnActivity(activities.PluginDeleteUserConnectionActivity, mock.Anything, mock.Anything).Return(
		nil, temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	// Mock task error update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSUConnection, DeletePSUConnection{
		TaskID:       taskID,
		ConnectorID:  s.connectorID,
		PsuID:        psuID,
		ConnectionID: connectionID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_DeletePSUConnection_ChildWorkflow_Error() {
	taskID := models.TaskID{
		Reference:   "test-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()
	connectionID := "test-connection-id"

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		CreatedAt: s.env.Now().UTC(),
	}

	connection := &models.OpenBankingConnection{
		ConnectionID:  connectionID,
		ConnectorID:   s.connectorID,
		CreatedAt:     s.env.Now().UTC(),
		DataUpdatedAt: s.env.Now().UTC(),
		Status:        models.ConnectionStatusActive,
		AccessToken: &models.Token{
			Token: "access-token",
		},
		Metadata: map[string]string{
			"bank": "test-bank",
		},
	}

	openBankingForwardedUser := &models.OpenBankingForwardedUser{
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

	// Mock connection retrieval
	s.env.OnActivity(activities.StorageOpenBankingConnectionsGetFromConnectionIDActivity, mock.Anything, s.connectorID, connectionID).Once().Return(
		&activities.StorageOpenBankingConnectionsGetFromConnectionIDResult{
			Connection: connection,
			PSUID:      psuID,
		}, nil,
	)

	// Mock open banking forwarded user retrieval
	s.env.OnActivity(activities.StorageOpenBankingForwardedUsersGetActivity, mock.Anything, psuID, s.connectorID).Once().Return(openBankingForwardedUser, nil)

	// Mock plugin delete user connection
	s.env.OnActivity(activities.PluginDeleteUserConnectionActivity, mock.Anything, mock.Anything).Return(&models.DeleteUserConnectionResponse{}, nil)

	// Mock child workflow execution error
	s.env.OnWorkflow(RunDeleteOpenBankingConnectionData, mock.Anything, mock.Anything).Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	// Mock task error update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSUConnection, DeletePSUConnection{
		TaskID:       taskID,
		ConnectorID:  s.connectorID,
		PsuID:        psuID,
		ConnectionID: connectionID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_DeletePSUConnection_StoragePSUOpenBankingConnectionDelete_Error() {
	taskID := models.TaskID{
		Reference:   "test-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()
	connectionID := "test-connection-id"

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		CreatedAt: s.env.Now().UTC(),
	}

	connection := &models.OpenBankingConnection{
		ConnectionID:  connectionID,
		ConnectorID:   s.connectorID,
		CreatedAt:     s.env.Now().UTC(),
		DataUpdatedAt: s.env.Now().UTC(),
		Status:        models.ConnectionStatusActive,
		AccessToken: &models.Token{
			Token: "access-token",
		},
		Metadata: map[string]string{
			"bank": "test-bank",
		},
	}

	openBankingForwardedUser := &models.OpenBankingForwardedUser{
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

	// Mock connection retrieval
	s.env.OnActivity(activities.StorageOpenBankingConnectionsGetFromConnectionIDActivity, mock.Anything, s.connectorID, connectionID).Once().Return(
		&activities.StorageOpenBankingConnectionsGetFromConnectionIDResult{
			Connection: connection,
			PSUID:      psuID,
		}, nil,
	)

	// Mock open banking forwarded user retrieval
	s.env.OnActivity(activities.StorageOpenBankingForwardedUsersGetActivity, mock.Anything, psuID, s.connectorID).Once().Return(openBankingForwardedUser, nil)

	// Mock plugin delete user connection
	s.env.OnActivity(activities.PluginDeleteUserConnectionActivity, mock.Anything, mock.Anything).Return(&models.DeleteUserConnectionResponse{}, nil)

	// Mock child workflow execution
	s.env.OnWorkflow(RunDeleteOpenBankingConnectionData, mock.Anything, mock.Anything).Return(nil)

	// Mock connection deletion error
	s.env.OnActivity(activities.StorageOpenBankingConnectionsDeleteActivity, mock.Anything, psuID, s.connectorID, connectionID).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	// Mock task error update
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(RunDeletePSUConnection, DeletePSUConnection{
		TaskID:       taskID,
		ConnectorID:  s.connectorID,
		PsuID:        psuID,
		ConnectionID: connectionID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_DeletePSUConnection_TaskErrorUpdate_Error() {
	taskID := models.TaskID{
		Reference:   "test-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()
	connectionID := "test-connection-id"

	// Mock PSU retrieval error
	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(
		nil, temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	// Mock task error update error
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("task-update-error", "task-update-error", errors.New("task-update-error")),
	)

	s.env.ExecuteWorkflow(RunDeletePSUConnection, DeletePSUConnection{
		TaskID:       taskID,
		ConnectorID:  s.connectorID,
		PsuID:        psuID,
		ConnectionID: connectionID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "task-update-error")
}

func (s *UnitTestSuite) Test_DeletePSUConnection_TaskSuccessUpdate_Error() {
	taskID := models.TaskID{
		Reference:   "test-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()
	connectionID := "test-connection-id"

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		CreatedAt: s.env.Now().UTC(),
	}

	connection := &models.OpenBankingConnection{
		ConnectionID:  connectionID,
		ConnectorID:   s.connectorID,
		CreatedAt:     s.env.Now().UTC(),
		DataUpdatedAt: s.env.Now().UTC(),
		Status:        models.ConnectionStatusActive,
		AccessToken: &models.Token{
			Token: "access-token",
		},
		Metadata: map[string]string{
			"bank": "test-bank",
		},
	}

	openBankingForwardedUser := &models.OpenBankingForwardedUser{
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

	// Mock connection retrieval
	s.env.OnActivity(activities.StorageOpenBankingConnectionsGetFromConnectionIDActivity, mock.Anything, s.connectorID, connectionID).Once().Return(
		&activities.StorageOpenBankingConnectionsGetFromConnectionIDResult{
			Connection: connection,
			PSUID:      psuID,
		}, nil,
	)

	// Mock open banking forwarded user retrieval
	s.env.OnActivity(activities.StorageOpenBankingForwardedUsersGetActivity, mock.Anything, psuID, s.connectorID).Once().Return(openBankingForwardedUser, nil)

	// Mock plugin delete user connection
	s.env.OnActivity(activities.PluginDeleteUserConnectionActivity, mock.Anything, mock.Anything).Return(&models.DeleteUserConnectionResponse{}, nil)

	// Mock child workflow execution
	s.env.OnWorkflow(RunDeleteOpenBankingConnectionData, mock.Anything, mock.Anything).Return(nil)

	// Mock connection deletion
	s.env.OnActivity(activities.StorageOpenBankingConnectionsDeleteActivity, mock.Anything, psuID, s.connectorID, connectionID).Once().Return(nil)

	// Mock task success update error
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("task-update-error", "task-update-error", errors.New("task-update-error")),
	)

	s.env.ExecuteWorkflow(RunDeletePSUConnection, DeletePSUConnection{
		TaskID:       taskID,
		ConnectorID:  s.connectorID,
		PsuID:        psuID,
		ConnectionID: connectionID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "task-update-error")
}
