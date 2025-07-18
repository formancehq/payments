package workflow

import (
	"context"
	"errors"
	"time"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_CreateUser_Success() {
	taskID := models.TaskID{
		Reference:   "create-user-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
		ContactDetails: &models.ContactDetails{
			Email:       stringPtr("test@example.com"),
			PhoneNumber: stringPtr("+1234567890"),
		},
		Address: &models.Address{
			StreetName:   stringPtr("Test Street"),
			StreetNumber: stringPtr("123"),
			City:         stringPtr("Test City"),
			Region:       stringPtr("Test Region"),
			PostalCode:   stringPtr("12345"),
			Country:      stringPtr("US"),
		},
		Metadata: map[string]string{
			"source": "test",
		},
	}

	createUserResponse := &models.CreateUserResponse{
		PermanentToken: &models.Token{
			Token:     "permanent-token-123",
			ExpiresAt: s.env.Now().UTC().Add(24 * time.Hour),
		},
		Metadata: map[string]string{
			"user_id": "external-user-123",
			"status":  "active",
		},
	}

	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)
	s.env.OnActivity(activities.PluginCreateUserActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CreateUserRequest) (*models.CreateUserResponse, error) {
		return createUserResponse, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgesStoreActivity, mock.Anything, psuID, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunCreateUser, CreateUser{
		TaskID:      taskID,
		ConnectorID: connectorID,
		PsuID:       psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_CreateUser_StoragePaymentServiceUsersGet_Error() {
	taskID := models.TaskID{
		Reference:   "create-user-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("user not found")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunCreateUser, CreateUser{
		TaskID:      taskID,
		ConnectorID: connectorID,
		PsuID:       psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "user not found")
}

func (s *UnitTestSuite) Test_CreateUser_PluginCreateUser_Error() {
	taskID := models.TaskID{
		Reference:   "create-user-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
		ContactDetails: &models.ContactDetails{
			Email:       stringPtr("test@example.com"),
			PhoneNumber: stringPtr("+1234567890"),
		},
		Metadata: map[string]string{
			"source": "test",
		},
	}

	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)
	s.env.OnActivity(activities.PluginCreateUserActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "PLUGIN", errors.New("plugin error")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunCreateUser, CreateUser{
		TaskID:      taskID,
		ConnectorID: connectorID,
		PsuID:       psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "plugin error")
}

func (s *UnitTestSuite) Test_CreateUser_StoragePSUBankBridgesStore_Error() {
	taskID := models.TaskID{
		Reference:   "create-user-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
		ContactDetails: &models.ContactDetails{
			Email:       stringPtr("test@example.com"),
			PhoneNumber: stringPtr("+1234567890"),
		},
		Metadata: map[string]string{
			"source": "test",
		},
	}

	createUserResponse := &models.CreateUserResponse{
		PermanentToken: &models.Token{
			Token:     "permanent-token-123",
			ExpiresAt: s.env.Now().UTC().Add(24 * time.Hour),
		},
		Metadata: map[string]string{
			"user_id": "external-user-123",
			"status":  "active",
		},
	}

	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)
	s.env.OnActivity(activities.PluginCreateUserActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CreateUserRequest) (*models.CreateUserResponse, error) {
		return createUserResponse, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgesStoreActivity, mock.Anything, psuID, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("storage error")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunCreateUser, CreateUser{
		TaskID:      taskID,
		ConnectorID: connectorID,
		PsuID:       psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "storage error")
}

func (s *UnitTestSuite) Test_CreateUser_UpdateTaskError_Error() {
	taskID := models.TaskID{
		Reference:   "create-user-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("user not found")),
	)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("task update error")),
	)

	s.env.ExecuteWorkflow(RunCreateUser, CreateUser{
		TaskID:      taskID,
		ConnectorID: connectorID,
		PsuID:       psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "task update error")
}

func (s *UnitTestSuite) Test_CreateUser_UpdateTaskSuccess_Error() {
	taskID := models.TaskID{
		Reference:   "create-user-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
		ContactDetails: &models.ContactDetails{
			Email:       stringPtr("test@example.com"),
			PhoneNumber: stringPtr("+1234567890"),
		},
		Metadata: map[string]string{
			"source": "test",
		},
	}

	createUserResponse := &models.CreateUserResponse{
		PermanentToken: &models.Token{
			Token:     "permanent-token-123",
			ExpiresAt: s.env.Now().UTC().Add(24 * time.Hour),
		},
		Metadata: map[string]string{
			"user_id": "external-user-123",
			"status":  "active",
		},
	}

	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)
	s.env.OnActivity(activities.PluginCreateUserActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CreateUserRequest) (*models.CreateUserResponse, error) {
		return createUserResponse, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgesStoreActivity, mock.Anything, psuID, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("task update error")),
	)

	s.env.ExecuteWorkflow(RunCreateUser, CreateUser{
		TaskID:      taskID,
		ConnectorID: connectorID,
		PsuID:       psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "task update error")
}

func (s *UnitTestSuite) Test_CreateUser_WithoutPermanentToken() {
	taskID := models.TaskID{
		Reference:   "create-user-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
		ContactDetails: &models.ContactDetails{
			Email:       stringPtr("test@example.com"),
			PhoneNumber: stringPtr("+1234567890"),
		},
		Metadata: map[string]string{
			"source": "test",
		},
	}

	createUserResponse := &models.CreateUserResponse{
		PermanentToken: nil, // No permanent token
		Metadata: map[string]string{
			"user_id": "external-user-123",
			"status":  "pending",
		},
	}

	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)
	s.env.OnActivity(activities.PluginCreateUserActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CreateUserRequest) (*models.CreateUserResponse, error) {
		return createUserResponse, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgesStoreActivity, mock.Anything, psuID, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunCreateUser, CreateUser{
		TaskID:      taskID,
		ConnectorID: connectorID,
		PsuID:       psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_CreateUser_WithMinimalPSU() {
	taskID := models.TaskID{
		Reference:   "create-user-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
		// No optional fields
	}

	createUserResponse := &models.CreateUserResponse{
		PermanentToken: &models.Token{
			Token:     "permanent-token-123",
			ExpiresAt: s.env.Now().UTC().Add(24 * time.Hour),
		},
		Metadata: map[string]string{
			"user_id": "external-user-123",
		},
	}

	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)
	s.env.OnActivity(activities.PluginCreateUserActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CreateUserRequest) (*models.CreateUserResponse, error) {
		return createUserResponse, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgesStoreActivity, mock.Anything, psuID, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunCreateUser, CreateUser{
		TaskID:      taskID,
		ConnectorID: connectorID,
		PsuID:       psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_CreateUser_WithEmptyMetadata() {
	taskID := models.TaskID{
		Reference:   "create-user-task",
		ConnectorID: s.connectorID,
	}
	psuID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	psu := &models.PaymentServiceUser{
		ID:        psuID,
		Name:      "Test User",
		CreatedAt: s.env.Now().UTC(),
		ContactDetails: &models.ContactDetails{
			Email:       stringPtr("test@example.com"),
			PhoneNumber: stringPtr("+1234567890"),
		},
		Metadata: map[string]string{}, // Empty metadata
	}

	createUserResponse := &models.CreateUserResponse{
		PermanentToken: &models.Token{
			Token:     "permanent-token-123",
			ExpiresAt: s.env.Now().UTC().Add(24 * time.Hour),
		},
		Metadata: map[string]string{}, // Empty metadata
	}

	s.env.OnActivity(activities.StoragePaymentServiceUsersGetActivity, mock.Anything, psuID).Once().Return(psu, nil)
	s.env.OnActivity(activities.PluginCreateUserActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CreateUserRequest) (*models.CreateUserResponse, error) {
		return createUserResponse, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgesStoreActivity, mock.Anything, psuID, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageTasksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunCreateUser, CreateUser{
		TaskID:      taskID,
		ConnectorID: connectorID,
		PsuID:       psuID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
