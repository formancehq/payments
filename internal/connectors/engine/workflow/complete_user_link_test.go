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

func (s *UnitTestSuite) Test_CompleteUserLink_Success() {
	attemptID := uuid.New()
	psuID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	attempt := &models.OpenBankingConnectionAttempt{
		ID:          attemptID,
		PsuID:       psuID,
		ConnectorID: connectorID,
		CreatedAt:   s.env.Now().UTC(),
		Status:      models.OpenBankingConnectionAttemptStatusPending,
	}

	connections := []models.PSPOpenBankingConnection{
		{
			ConnectionID: "conn-1",
			CreatedAt:    s.env.Now().UTC(),
			AccessToken: &models.Token{
				Token: "access-token-1",
			},
			Metadata: map[string]string{
				"bank": "test-bank",
			},
		},
		{
			ConnectionID: "conn-2",
			CreatedAt:    s.env.Now().UTC(),
			AccessToken: &models.Token{
				Token: "access-token-2",
			},
			Metadata: map[string]string{
				"bank": "test-bank-2",
			},
		},
	}

	httpCallInfo := models.HTTPCallInformation{
		QueryValues: map[string][]string{
			"code":  {"auth-code"},
			"state": {"callback-state"},
		},
		Headers: map[string][]string{
			"authorization": {"Bearer token"},
		},
		Body: []byte("test body"),
	}

	s.env.OnActivity(activities.StorageOpenBankingConnectionAttemptsGetActivity, mock.Anything, attemptID).Once().Return(attempt, nil)
	s.env.OnActivity(activities.PluginCompleteUserLinkActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CompleteUserLinkRequest) (*models.CompleteUserLinkResponse, error) {
		return &models.CompleteUserLinkResponse{
			Success: &models.UserLinkSuccessResponse{
				Connections: connections,
			},
		}, nil
	})
	s.env.OnActivity(activities.StorageOpenBankingConnectionAttemptsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageOpenBankingConnectionsStoreActivity, mock.Anything, psuID, mock.Anything).Times(2).Return(nil)

	s.env.ExecuteWorkflow(RunCompleteUserLink, CompleteUserLink{
		HTTPCallInformation: httpCallInfo,
		ConnectorID:         connectorID,
		AttemptID:           attemptID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_CompleteUserLink_StoragePSUOpenBankingConnectionAttemptsGet_Error() {
	attemptID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	httpCallInfo := models.HTTPCallInformation{
		QueryValues: map[string][]string{
			"code": {"auth-code"},
		},
	}

	s.env.OnActivity(activities.StorageOpenBankingConnectionAttemptsGetActivity, mock.Anything, attemptID).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("attempt not found")),
	)

	s.env.ExecuteWorkflow(RunCompleteUserLink, CompleteUserLink{
		HTTPCallInformation: httpCallInfo,
		ConnectorID:         connectorID,
		AttemptID:           attemptID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "attempt not found")
}

func (s *UnitTestSuite) Test_CompleteUserLink_PluginCompleteUserLink_Error() {
	attemptID := uuid.New()
	psuID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	attempt := &models.OpenBankingConnectionAttempt{
		ID:          attemptID,
		PsuID:       psuID,
		ConnectorID: connectorID,
		CreatedAt:   s.env.Now().UTC(),
		Status:      models.OpenBankingConnectionAttemptStatusPending,
	}

	httpCallInfo := models.HTTPCallInformation{
		QueryValues: map[string][]string{
			"code": {"auth-code"},
		},
	}

	s.env.OnActivity(activities.StorageOpenBankingConnectionAttemptsGetActivity, mock.Anything, attemptID).Once().Return(attempt, nil)
	s.env.OnActivity(activities.PluginCompleteUserLinkActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "PLUGIN", errors.New("plugin error")),
	)

	s.env.ExecuteWorkflow(RunCompleteUserLink, CompleteUserLink{
		HTTPCallInformation: httpCallInfo,
		ConnectorID:         connectorID,
		AttemptID:           attemptID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "plugin error")
}

func (s *UnitTestSuite) Test_CompleteUserLink_PluginErrorResponse() {
	attemptID := uuid.New()
	psuID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	attempt := &models.OpenBankingConnectionAttempt{
		ID:          attemptID,
		PsuID:       psuID,
		ConnectorID: connectorID,
		CreatedAt:   s.env.Now().UTC(),
		Status:      models.OpenBankingConnectionAttemptStatusPending,
	}

	httpCallInfo := models.HTTPCallInformation{
		QueryValues: map[string][]string{
			"code": {"auth-code"},
		},
	}

	s.env.OnActivity(activities.StorageOpenBankingConnectionAttemptsGetActivity, mock.Anything, attemptID).Once().Return(attempt, nil)
	s.env.OnActivity(activities.PluginCompleteUserLinkActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CompleteUserLinkRequest) (*models.CompleteUserLinkResponse, error) {
		return &models.CompleteUserLinkResponse{
			Error: &models.UserLinkErrorResponse{
				Error: "authentication failed",
			},
		}, nil
	})
	s.env.OnActivity(activities.StorageOpenBankingConnectionAttemptsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunCompleteUserLink, CompleteUserLink{
		HTTPCallInformation: httpCallInfo,
		ConnectorID:         connectorID,
		AttemptID:           attemptID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_CompleteUserLink_UnexpectedResponse() {
	attemptID := uuid.New()
	psuID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	attempt := &models.OpenBankingConnectionAttempt{
		ID:          attemptID,
		PsuID:       psuID,
		ConnectorID: connectorID,
		CreatedAt:   s.env.Now().UTC(),
		Status:      models.OpenBankingConnectionAttemptStatusPending,
	}

	httpCallInfo := models.HTTPCallInformation{
		QueryValues: map[string][]string{
			"code": {"auth-code"},
		},
	}

	s.env.OnActivity(activities.StorageOpenBankingConnectionAttemptsGetActivity, mock.Anything, attemptID).Once().Return(attempt, nil)
	s.env.OnActivity(activities.PluginCompleteUserLinkActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CompleteUserLinkRequest) (*models.CompleteUserLinkResponse, error) {
		return &models.CompleteUserLinkResponse{
			Success: nil,
			Error:   nil,
		}, nil
	})
	s.env.OnActivity(activities.StorageOpenBankingConnectionAttemptsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunCompleteUserLink, CompleteUserLink{
		HTTPCallInformation: httpCallInfo,
		ConnectorID:         connectorID,
		AttemptID:           attemptID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_CompleteUserLink_StoragePSUOpenBankingConnectionAttemptsStore_Error_OnFailure() {
	attemptID := uuid.New()
	psuID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	attempt := &models.OpenBankingConnectionAttempt{
		ID:          attemptID,
		PsuID:       psuID,
		ConnectorID: connectorID,
		CreatedAt:   s.env.Now().UTC(),
		Status:      models.OpenBankingConnectionAttemptStatusPending,
	}

	httpCallInfo := models.HTTPCallInformation{
		QueryValues: map[string][]string{
			"code": {"auth-code"},
		},
	}

	s.env.OnActivity(activities.StorageOpenBankingConnectionAttemptsGetActivity, mock.Anything, attemptID).Once().Return(attempt, nil)
	s.env.OnActivity(activities.PluginCompleteUserLinkActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CompleteUserLinkRequest) (*models.CompleteUserLinkResponse, error) {
		return &models.CompleteUserLinkResponse{
			Error: &models.UserLinkErrorResponse{
				Error: "authentication failed",
			},
		}, nil
	})
	s.env.OnActivity(activities.StorageOpenBankingConnectionAttemptsStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("storage error")),
	)

	s.env.ExecuteWorkflow(RunCompleteUserLink, CompleteUserLink{
		HTTPCallInformation: httpCallInfo,
		ConnectorID:         connectorID,
		AttemptID:           attemptID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "storage error")
}

func (s *UnitTestSuite) Test_CompleteUserLink_StoragePSUOpenBankingConnectionAttemptsStore_Error_OnSuccess() {
	attemptID := uuid.New()
	psuID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	attempt := &models.OpenBankingConnectionAttempt{
		ID:          attemptID,
		PsuID:       psuID,
		ConnectorID: connectorID,
		CreatedAt:   s.env.Now().UTC(),
		Status:      models.OpenBankingConnectionAttemptStatusPending,
	}

	connections := []models.PSPOpenBankingConnection{
		{
			ConnectionID: "conn-1",
			CreatedAt:    s.env.Now().UTC(),
			AccessToken: &models.Token{
				Token: "access-token-1",
			},
		},
	}

	httpCallInfo := models.HTTPCallInformation{
		QueryValues: map[string][]string{
			"code": {"auth-code"},
		},
	}

	s.env.OnActivity(activities.StorageOpenBankingConnectionAttemptsGetActivity, mock.Anything, attemptID).Once().Return(attempt, nil)
	s.env.OnActivity(activities.PluginCompleteUserLinkActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CompleteUserLinkRequest) (*models.CompleteUserLinkResponse, error) {
		return &models.CompleteUserLinkResponse{
			Success: &models.UserLinkSuccessResponse{
				Connections: connections,
			},
		}, nil
	})
	s.env.OnActivity(activities.StorageOpenBankingConnectionAttemptsStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("storage error")),
	)

	s.env.ExecuteWorkflow(RunCompleteUserLink, CompleteUserLink{
		HTTPCallInformation: httpCallInfo,
		ConnectorID:         connectorID,
		AttemptID:           attemptID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "storage error")
}

func (s *UnitTestSuite) Test_CompleteUserLink_StoragePSUOpenBankingConnectionsStore_Error() {
	attemptID := uuid.New()
	psuID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	attempt := &models.OpenBankingConnectionAttempt{
		ID:          attemptID,
		PsuID:       psuID,
		ConnectorID: connectorID,
		CreatedAt:   s.env.Now().UTC(),
		Status:      models.OpenBankingConnectionAttemptStatusPending,
	}

	connections := []models.PSPOpenBankingConnection{
		{
			ConnectionID: "conn-1",
			CreatedAt:    s.env.Now().UTC(),
			AccessToken: &models.Token{
				Token: "access-token-1",
			},
		},
	}

	httpCallInfo := models.HTTPCallInformation{
		QueryValues: map[string][]string{
			"code": {"auth-code"},
		},
	}

	s.env.OnActivity(activities.StorageOpenBankingConnectionAttemptsGetActivity, mock.Anything, attemptID).Once().Return(attempt, nil)
	s.env.OnActivity(activities.PluginCompleteUserLinkActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CompleteUserLinkRequest) (*models.CompleteUserLinkResponse, error) {
		return &models.CompleteUserLinkResponse{
			Success: &models.UserLinkSuccessResponse{
				Connections: connections,
			},
		}, nil
	})
	s.env.OnActivity(activities.StorageOpenBankingConnectionAttemptsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageOpenBankingConnectionsStoreActivity, mock.Anything, psuID, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("connection storage error")),
	)

	s.env.ExecuteWorkflow(RunCompleteUserLink, CompleteUserLink{
		HTTPCallInformation: httpCallInfo,
		ConnectorID:         connectorID,
		AttemptID:           attemptID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "connection storage error")
}

func (s *UnitTestSuite) Test_CompleteUserLink_EmptyConnections() {
	attemptID := uuid.New()
	psuID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	attempt := &models.OpenBankingConnectionAttempt{
		ID:          attemptID,
		PsuID:       psuID,
		ConnectorID: connectorID,
		CreatedAt:   s.env.Now().UTC(),
		Status:      models.OpenBankingConnectionAttemptStatusPending,
	}

	httpCallInfo := models.HTTPCallInformation{
		QueryValues: map[string][]string{
			"code": {"auth-code"},
		},
	}

	s.env.OnActivity(activities.StorageOpenBankingConnectionAttemptsGetActivity, mock.Anything, attemptID).Once().Return(attempt, nil)
	s.env.OnActivity(activities.PluginCompleteUserLinkActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CompleteUserLinkRequest) (*models.CompleteUserLinkResponse, error) {
		return &models.CompleteUserLinkResponse{
			Success: &models.UserLinkSuccessResponse{
				Connections: []models.PSPOpenBankingConnection{},
			},
		}, nil
	})
	s.env.OnActivity(activities.StorageOpenBankingConnectionAttemptsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunCompleteUserLink, CompleteUserLink{
		HTTPCallInformation: httpCallInfo,
		ConnectorID:         connectorID,
		AttemptID:           attemptID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_CompleteUserLink_PluginErrorWithEmptyError() {
	attemptID := uuid.New()
	psuID := uuid.New()
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	attempt := &models.OpenBankingConnectionAttempt{
		ID:          attemptID,
		PsuID:       psuID,
		ConnectorID: connectorID,
		CreatedAt:   s.env.Now().UTC(),
		Status:      models.OpenBankingConnectionAttemptStatusPending,
	}

	httpCallInfo := models.HTTPCallInformation{
		QueryValues: map[string][]string{
			"code": {"auth-code"},
		},
	}

	s.env.OnActivity(activities.StorageOpenBankingConnectionAttemptsGetActivity, mock.Anything, attemptID).Once().Return(attempt, nil)
	s.env.OnActivity(activities.PluginCompleteUserLinkActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CompleteUserLinkRequest) (*models.CompleteUserLinkResponse, error) {
		return &models.CompleteUserLinkResponse{
			Error: &models.UserLinkErrorResponse{
				Error: "",
			},
		}, nil
	})
	s.env.OnActivity(activities.StorageOpenBankingConnectionAttemptsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunCompleteUserLink, CompleteUserLink{
		HTTPCallInformation: httpCallInfo,
		ConnectorID:         connectorID,
		AttemptID:           attemptID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}
