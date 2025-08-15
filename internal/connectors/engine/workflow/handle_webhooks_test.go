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
	"go.temporal.io/sdk/workflow"
)

func (s *UnitTestSuite) Test_HandleWebhooks_Success() {
	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, webhook models.Webhook) error {
		return nil
	})
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
		return &models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{
				{
					Account:         &s.pspAccount,
					ExternalAccount: &s.pspAccount,
					Payment:         &s.pspPayment,
				},
			},
		}, nil
	})
	s.env.OnWorkflow(RunStoreWebhookTranslation, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req StoreWebhookTranslation) error {
		s.Equal(s.connectorID, req.ConnectorID)
		s.NotNil(req.Account)
		s.Equal(s.accountID.Reference, req.Account.Reference)
		s.NotNil(req.ExternalAccount)
		s.Equal(s.accountID.Reference, req.ExternalAccount.Reference)
		s.NotNil(req.Payment)
		s.Equal(s.paymentPayoutID.Reference, req.Payment.Reference)
		return nil
	})

	s.env.ExecuteWorkflow(RunHandleWebhooks, HandleWebhooks{
		ConnectorID: s.connectorID,
		URLPath:     "/test",
		Webhook: models.Webhook{
			ID:          "test",
			ConnectorID: s.connectorID,
			QueryValues: map[string][]string{
				"test": {"test"},
			},
			Headers: map[string][]string{
				"test": {"test"},
			},
			Body: []byte(`{}`),
		},
		Config: &models.WebhookConfig{
			Name:        "test",
			ConnectorID: s.connectorID,
			URLPath:     "/test",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_HandleWebhooks_NoResponses_Success() {
	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, webhook models.Webhook) error {
		return nil
	})
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
		return &models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{},
		}, nil
	})

	s.env.ExecuteWorkflow(RunHandleWebhooks, HandleWebhooks{
		ConnectorID: s.connectorID,
		URLPath:     "/test",
		Webhook: models.Webhook{
			ID:          "test",
			ConnectorID: s.connectorID,
			QueryValues: map[string][]string{
				"test": {"test"},
			},
			Headers: map[string][]string{
				"test": {"test"},
			},
			Body: []byte(`{}`),
		},
		Config: &models.WebhookConfig{
			Name:        "test",
			ConnectorID: s.connectorID,
			URLPath:     "/test",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_HandleWebhooks_StorageWebhooksStore_Error() {
	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunHandleWebhooks, HandleWebhooks{
		ConnectorID: s.connectorID,
		URLPath:     "/test",
		Webhook: models.Webhook{
			ID:          "test",
			ConnectorID: s.connectorID,
			QueryValues: map[string][]string{
				"test": {"test"},
			},
			Headers: map[string][]string{
				"test": {"test"},
			},
			Body: []byte(`{}`),
		},
		Config: &models.WebhookConfig{
			Name:        "test",
			ConnectorID: s.connectorID,
			URLPath:     "/test",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_HandleWebhooks_PluginTranslateWebhook_Error() {
	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(nil,
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunHandleWebhooks, HandleWebhooks{
		ConnectorID: s.connectorID,
		URLPath:     "/test",
		Webhook: models.Webhook{
			ID:          "test",
			ConnectorID: s.connectorID,
			QueryValues: map[string][]string{
				"test": {"test"},
			},
			Headers: map[string][]string{
				"test": {"test"},
			},
			Body: []byte(`{}`),
		},
		Config: &models.WebhookConfig{
			Name:        "test",
			ConnectorID: s.connectorID,
			URLPath:     "/test",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_HandleWebhooks_RunStoreWebhookTranslation_Error() {
	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
		return &models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{
				{
					Account:         &s.pspAccount,
					ExternalAccount: &s.pspAccount,
					Payment:         &s.pspPayment,
				},
			},
		}, nil
	})
	s.env.OnWorkflow(RunStoreWebhookTranslation, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunHandleWebhooks, HandleWebhooks{
		ConnectorID: s.connectorID,
		URLPath:     "/test",
		Webhook: models.Webhook{
			ID:          "test",
			ConnectorID: s.connectorID,
			QueryValues: map[string][]string{
				"test": {"test"},
			},
			Headers: map[string][]string{
				"test": {"test"},
			},
			Body: []byte(`{}`),
		},
		Config: &models.WebhookConfig{
			Name:        "test",
			ConnectorID: s.connectorID,
			URLPath:     "/test",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

// DataReadyToFetch webhook tests
func (s *UnitTestSuite) Test_HandleWebhooks_DataReadyToFetch_Success() {
	connectionID := "test-connection-id"
	psuID := uuid.New()

	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
		return &models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{
				{
					DataReadyToFetch: &models.PSPDataReadyToFetch{
						ConnectionID: &connectionID,
						FromPayload:  []byte(`{"test": "data"}`),
					},
				},
			},
		}, nil
	})
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, connectorID models.ConnectorID) (*models.Connector, error) {
		return &s.connector, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDActivity, mock.Anything, mock.Anything, mock.Anything).Return(func(ctx context.Context, connectorID models.ConnectorID, connectionID string) (*activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDResult, error) {
		return &activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDResult{
			Connection: &models.PSUBankBridgeConnection{
				ConnectionID: connectionID,
				ConnectorID:  s.connectorID,
				CreatedAt:    time.Now(),
				Status:       models.ConnectionStatusActive,
			},
			PSUID: psuID,
		}, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgesGetActivity, mock.Anything, mock.Anything, mock.Anything).Return(func(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) (*models.PSUBankBridge, error) {
		return &models.PSUBankBridge{
			ConnectorID: connectorID,
			AccessToken: &models.Token{
				Token:     "test-token",
				ExpiresAt: time.Now().Add(time.Hour),
			},
		}, nil
	})
	s.env.OnWorkflow(RunFetchBankBridgeData, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req FetchBankBridgeData, tasks []models.ConnectorTaskTree) error {
		s.Equal(psuID, req.PsuID)
		s.Equal(connectionID, req.ConnectionID)
		s.Equal(s.connectorID, req.ConnectorID)
		s.NotNil(req.FromPayload)
		s.Equal(connectionID, req.FromPayload.ID)
		return nil
	})

	s.env.ExecuteWorkflow(RunHandleWebhooks, HandleWebhooks{
		ConnectorID: s.connectorID,
		URL:         "https://example.com/webhook",
		URLPath:     "/webhook",
		Webhook: models.Webhook{
			ID:          "test-webhook",
			ConnectorID: s.connectorID,
			Body:        []byte(`{"test": "data"}`),
		},
		Config: &models.WebhookConfig{
			Name:        "test",
			ConnectorID: s.connectorID,
			URLPath:     "/test",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_HandleWebhooks_DataReadyToFetch_StorageConnectorsGet_Error() {
	connectionID := "test-connection-id"

	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
		return &models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{
				{
					DataReadyToFetch: &models.PSPDataReadyToFetch{
						ConnectionID: &connectionID,
						FromPayload:  []byte(`{"test": "data"}`),
					},
				},
			},
		}, nil
	})
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, mock.Anything).Once().Return(
		(*models.Connector)(nil), temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunHandleWebhooks, HandleWebhooks{
		ConnectorID: s.connectorID,
		URLPath:     "/test",
		Webhook: models.Webhook{
			ID:          "test",
			ConnectorID: s.connectorID,
			QueryValues: map[string][]string{
				"test": {"test"},
			},
			Headers: map[string][]string{
				"test": {"test"},
			},
			Body: []byte(`{}`),
		},
		Config: &models.WebhookConfig{
			Name:        "test",
			ConnectorID: s.connectorID,
			URLPath:     "/test",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

// UserLinkSessionFinished webhook tests
func (s *UnitTestSuite) Test_HandleWebhooks_UserLinkSessionFinished_Success() {
	attemptID := uuid.New()
	status := models.PSUBankBridgeConnectionAttemptStatusCompleted
	errorMsg := "test error"

	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
		return &models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{
				{
					UserLinkSessionFinished: &models.PSPUserLinkSessionFinished{
						AttemptID: attemptID,
						Status:    status,
						Error:     &errorMsg,
					},
				},
			},
		}, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionAttemptsGetActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, attemptID uuid.UUID) (*models.PSUBankBridgeConnectionAttempt, error) {
		return &models.PSUBankBridgeConnectionAttempt{
			ID:          attemptID,
			PsuID:       uuid.New(),
			ConnectorID: s.connectorID,
		}, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionAttemptsUpdateStatusActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.UserLinkStatus)
		s.Equal(attemptID, req.UserLinkStatus.AttemptID)
		s.Equal(status, req.UserLinkStatus.Status)
		s.Equal(&errorMsg, req.UserLinkStatus.Error)
		return nil
	})

	s.env.ExecuteWorkflow(RunHandleWebhooks, HandleWebhooks{
		ConnectorID: s.connectorID,
		URLPath:     "/test",
		Webhook: models.Webhook{
			ID:          "test",
			ConnectorID: s.connectorID,
			QueryValues: map[string][]string{
				"test": {"test"},
			},
			Headers: map[string][]string{
				"test": {"test"},
			},
			Body: []byte(`{}`),
		},
		Config: &models.WebhookConfig{
			Name:        "test",
			ConnectorID: s.connectorID,
			URLPath:     "/test",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_HandleWebhooks_UserLinkSessionFinished_StoragePSUBankBridgeConnectionAttemptsGet_Error() {
	attemptID := uuid.New()

	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
		return &models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{
				{
					UserLinkSessionFinished: &models.PSPUserLinkSessionFinished{
						AttemptID: attemptID,
						Status:    models.PSUBankBridgeConnectionAttemptStatusCompleted,
					},
				},
			},
		}, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionAttemptsGetActivity, mock.Anything, mock.Anything).Return(
		(*models.PSUBankBridgeConnectionAttempt)(nil), temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunHandleWebhooks, HandleWebhooks{
		ConnectorID: s.connectorID,
		URLPath:     "/test",
		Webhook: models.Webhook{
			ID:          "test",
			ConnectorID: s.connectorID,
			QueryValues: map[string][]string{
				"test": {"test"},
			},
			Headers: map[string][]string{
				"test": {"test"},
			},
			Body: []byte(`{}`),
		},
		Config: &models.WebhookConfig{
			Name:        "test",
			ConnectorID: s.connectorID,
			URLPath:     "/test",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

// UserConnectionPendingDisconnect webhook tests
func (s *UnitTestSuite) Test_HandleWebhooks_UserConnectionPendingDisconnect_Success() {
	connectionID := "test-connection-id"
	psuID := uuid.New()
	reason := "test reason"
	at := time.Now()

	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
		return &models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{
				{
					UserConnectionPendingDisconnect: &models.PSPUserConnectionPendingDisconnect{
						ConnectionID: connectionID,
						At:           at,
						Reason:       &reason,
					},
				},
			},
		}, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDActivity, mock.Anything, mock.Anything, mock.Anything).Return(func(ctx context.Context, connectorID models.ConnectorID, connectionID string) (*activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDResult, error) {
		return &activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDResult{
			Connection: &models.PSUBankBridgeConnection{
				ConnectionID: connectionID,
				ConnectorID:  s.connectorID,
				Status:       models.ConnectionStatusActive,
			},
			PSUID: psuID,
		}, nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.UserPendingDisconnect)
		s.Equal(psuID, req.UserPendingDisconnect.PsuID)
		s.Equal(s.connectorID, req.UserPendingDisconnect.ConnectorID)
		s.Equal(connectionID, req.UserPendingDisconnect.ConnectionID)
		s.Equal(&reason, req.UserPendingDisconnect.Reason)
		return nil
	})

	s.env.ExecuteWorkflow(RunHandleWebhooks, HandleWebhooks{
		ConnectorID: s.connectorID,
		URLPath:     "/test",
		Webhook: models.Webhook{
			ID:          "test",
			ConnectorID: s.connectorID,
			QueryValues: map[string][]string{
				"test": {"test"},
			},
			Headers: map[string][]string{
				"test": {"test"},
			},
			Body: []byte(`{}`),
		},
		Config: &models.WebhookConfig{
			Name:        "test",
			ConnectorID: s.connectorID,
			URLPath:     "/test",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_HandleWebhooks_UserConnectionPendingDisconnect_StoragePSUBankBridgeConnectionsGetFromConnectionID_Error() {
	connectionID := "test-connection-id"

	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
		return &models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{
				{
					UserConnectionPendingDisconnect: &models.PSPUserConnectionPendingDisconnect{
						ConnectionID: connectionID,
						At:           time.Now(),
					},
				},
			},
		}, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		(*activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDResult)(nil), temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunHandleWebhooks, HandleWebhooks{
		ConnectorID: s.connectorID,
		URLPath:     "/test",
		Webhook: models.Webhook{
			ID:          "test",
			ConnectorID: s.connectorID,
			QueryValues: map[string][]string{
				"test": {"test"},
			},
			Headers: map[string][]string{
				"test": {"test"},
			},
			Body: []byte(`{}`),
		},
		Config: &models.WebhookConfig{
			Name:        "test",
			ConnectorID: s.connectorID,
			URLPath:     "/test",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

// UserConnectionDisconnected webhook tests
func (s *UnitTestSuite) Test_HandleWebhooks_UserConnectionDisconnected_Success() {
	connectionID := "test-connection-id"
	psuID := uuid.New()
	reason := "test reason"
	at := time.Now()

	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
		return &models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{
				{
					UserConnectionDisconnected: &models.PSPUserConnectionDisconnected{
						ConnectionID: connectionID,
						At:           at,
						Reason:       &reason,
					},
				},
			},
		}, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDActivity, mock.Anything, mock.Anything, mock.Anything).Return(func(ctx context.Context, connectorID models.ConnectorID, connectionID string) (*activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDResult, error) {
		return &activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDResult{
			Connection: &models.PSUBankBridgeConnection{
				ConnectionID: connectionID,
				ConnectorID:  s.connectorID,
				Status:       models.ConnectionStatusActive,
			},
			PSUID: psuID,
		}, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionsStoreActivity, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.UserConnectionDisconnected)
		s.Equal(psuID, req.UserConnectionDisconnected.PsuID)
		s.Equal(s.connectorID, req.UserConnectionDisconnected.ConnectorID)
		s.Equal(connectionID, req.UserConnectionDisconnected.ConnectionID)
		s.Equal(&reason, req.UserConnectionDisconnected.Reason)
		return nil
	})

	s.env.ExecuteWorkflow(RunHandleWebhooks, HandleWebhooks{
		ConnectorID: s.connectorID,
		URLPath:     "/test",
		Webhook: models.Webhook{
			ID:          "test",
			ConnectorID: s.connectorID,
			QueryValues: map[string][]string{
				"test": {"test"},
			},
			Headers: map[string][]string{
				"test": {"test"},
			},
			Body: []byte(`{}`),
		},
		Config: &models.WebhookConfig{
			Name:        "test",
			ConnectorID: s.connectorID,
			URLPath:     "/test",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_HandleWebhooks_UserConnectionDisconnected_StoragePSUBankBridgeConnectionsGetFromConnectionID_Error() {
	connectionID := "test-connection-id"

	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
		return &models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{
				{
					UserConnectionDisconnected: &models.PSPUserConnectionDisconnected{
						ConnectionID: connectionID,
						At:           time.Now(),
					},
				},
			},
		}, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		(*activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDResult)(nil), temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunHandleWebhooks, HandleWebhooks{
		ConnectorID: s.connectorID,
		URLPath:     "/test",
		Webhook: models.Webhook{
			ID:          "test",
			ConnectorID: s.connectorID,
			QueryValues: map[string][]string{
				"test": {"test"},
			},
			Headers: map[string][]string{
				"test": {"test"},
			},
			Body: []byte(`{}`),
		},
		Config: &models.WebhookConfig{
			Name:        "test",
			ConnectorID: s.connectorID,
			URLPath:     "/test",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_HandleWebhooks_UserConnectionDisconnected_StoragePSUBankBridgeConnectionsStore_Error() {
	connectionID := "test-connection-id"
	psuID := uuid.New()
	reason := "test reason"
	at := time.Now()

	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
		return &models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{
				{
					UserConnectionDisconnected: &models.PSPUserConnectionDisconnected{
						ConnectionID: connectionID,
						At:           at,
						Reason:       &reason,
					},
				},
			},
		}, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDActivity, mock.Anything, mock.Anything, mock.Anything).Return(func(ctx context.Context, connectorID models.ConnectorID, connectionID string) (*activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDResult, error) {
		return &activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDResult{
			Connection: &models.PSUBankBridgeConnection{
				ConnectionID: connectionID,
				ConnectorID:  s.connectorID,
				Status:       models.ConnectionStatusActive,
			},
			PSUID: psuID,
		}, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionsStoreActivity, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunHandleWebhooks, HandleWebhooks{
		ConnectorID: s.connectorID,
		URLPath:     "/test",
		Webhook: models.Webhook{
			ID:          "test",
			ConnectorID: s.connectorID,
			QueryValues: map[string][]string{
				"test": {"test"},
			},
			Headers: map[string][]string{
				"test": {"test"},
			},
			Body: []byte(`{}`),
		},
		Config: &models.WebhookConfig{
			Name:        "test",
			ConnectorID: s.connectorID,
			URLPath:     "/test",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

// UserConnectionReconnected webhook tests
func (s *UnitTestSuite) Test_HandleWebhooks_UserConnectionReconnected_Success() {
	connectionID := "test-connection-id"
	psuID := uuid.New()
	at := time.Now()

	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
		return &models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{
				{
					UserConnectionReconnected: &models.PSPUserConnectionReconnected{
						ConnectionID: connectionID,
						At:           at,
					},
				},
			},
		}, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDActivity, mock.Anything, mock.Anything, mock.Anything).Return(func(ctx context.Context, connectorID models.ConnectorID, connectionID string) (*activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDResult, error) {
		return &activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDResult{
			Connection: &models.PSUBankBridgeConnection{
				ConnectionID: connectionID,
				ConnectorID:  s.connectorID,
				Status:       models.ConnectionStatusActive,
			},
			PSUID: psuID,
		}, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionsStoreActivity, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.UserConnectionReconnected)
		s.Equal(psuID, req.UserConnectionReconnected.PsuID)
		s.Equal(s.connectorID, req.UserConnectionReconnected.ConnectorID)
		s.Equal(connectionID, req.UserConnectionReconnected.ConnectionID)
		return nil
	})

	s.env.ExecuteWorkflow(RunHandleWebhooks, HandleWebhooks{
		ConnectorID: s.connectorID,
		URLPath:     "/test",
		Webhook: models.Webhook{
			ID:          "test",
			ConnectorID: s.connectorID,
			QueryValues: map[string][]string{
				"test": {"test"},
			},
			Headers: map[string][]string{
				"test": {"test"},
			},
			Body: []byte(`{}`),
		},
		Config: &models.WebhookConfig{
			Name:        "test",
			ConnectorID: s.connectorID,
			URLPath:     "/test",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_HandleWebhooks_UserConnectionReconnected_StoragePSUBankBridgeConnectionsGetFromConnectionID_Error() {
	connectionID := "test-connection-id"

	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
		return &models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{
				{
					UserConnectionReconnected: &models.PSPUserConnectionReconnected{
						ConnectionID: connectionID,
						At:           time.Now(),
					},
				},
			},
		}, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		(*activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDResult)(nil), temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunHandleWebhooks, HandleWebhooks{
		ConnectorID: s.connectorID,
		URLPath:     "/test",
		Webhook: models.Webhook{
			ID:          "test",
			ConnectorID: s.connectorID,
			QueryValues: map[string][]string{
				"test": {"test"},
			},
			Headers: map[string][]string{
				"test": {"test"},
			},
			Body: []byte(`{}`),
		},
		Config: &models.WebhookConfig{
			Name:        "test",
			ConnectorID: s.connectorID,
			URLPath:     "/test",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

func (s *UnitTestSuite) Test_HandleWebhooks_UserConnectionReconnected_StoragePSUBankBridgeConnectionsStore_Error() {
	connectionID := "test-connection-id"
	psuID := uuid.New()
	at := time.Now()

	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
		return &models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{
				{
					UserConnectionReconnected: &models.PSPUserConnectionReconnected{
						ConnectionID: connectionID,
						At:           at,
					},
				},
			},
		}, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDActivity, mock.Anything, mock.Anything, mock.Anything).Return(func(ctx context.Context, connectorID models.ConnectorID, connectionID string) (*activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDResult, error) {
		return &activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDResult{
			Connection: &models.PSUBankBridgeConnection{
				ConnectionID: connectionID,
				ConnectorID:  s.connectorID,
				Status:       models.ConnectionStatusActive,
			},
			PSUID: psuID,
		}, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionsStoreActivity, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("error-test", "error-test", errors.New("error-test")),
	)

	s.env.ExecuteWorkflow(RunHandleWebhooks, HandleWebhooks{
		ConnectorID: s.connectorID,
		URLPath:     "/test",
		Webhook: models.Webhook{
			ID:          "test",
			ConnectorID: s.connectorID,
			QueryValues: map[string][]string{
				"test": {"test"},
			},
			Headers: map[string][]string{
				"test": {"test"},
			},
			Body: []byte(`{}`),
		},
		Config: &models.WebhookConfig{
			Name:        "test",
			ConnectorID: s.connectorID,
			URLPath:     "/test",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "error-test")
}

// Multiple webhook responses test
func (s *UnitTestSuite) Test_HandleWebhooks_MultipleResponses_Success() {
	connectionID := "test-connection-id"
	attemptID := uuid.New()

	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
		return &models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{
				{
					Account: &s.pspAccount,
				},
				{
					DataReadyToFetch: &models.PSPDataReadyToFetch{
						ConnectionID: &connectionID,
						FromPayload:  []byte(`{"test": "data"}`),
					},
				},
				{
					UserLinkSessionFinished: &models.PSPUserLinkSessionFinished{
						AttemptID: attemptID,
						Status:    models.PSUBankBridgeConnectionAttemptStatusCompleted,
					},
				},
			},
		}, nil
	})

	// Mock for DataReadyToFetch
	s.env.OnActivity(activities.StorageConnectorsGetActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, connectorID models.ConnectorID) (*models.Connector, error) {
		return &s.connector, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDActivity, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, connectorID models.ConnectorID, connectionID string) (*activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDResult, error) {
		return &activities.StoragePSUBankBridgeConnectionsGetFromConnectionIDResult{
			Connection: &models.PSUBankBridgeConnection{
				ConnectionID: connectionID,
				ConnectorID:  s.connectorID,
				Status:       models.ConnectionStatusActive,
			},
			PSUID: uuid.New(),
		}, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgesGetActivity, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) (*models.PSUBankBridge, error) {
		return &models.PSUBankBridge{
			ConnectorID: connectorID,
		}, nil
	})
	s.env.OnWorkflow(RunFetchBankBridgeData, mock.Anything, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req FetchBankBridgeData, tasks []models.ConnectorTaskTree) error {
		return nil
	})

	// Mock for UserLinkSessionFinished
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionAttemptsGetActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, attemptID uuid.UUID) (*models.PSUBankBridgeConnectionAttempt, error) {
		return &models.PSUBankBridgeConnectionAttempt{
			ID:          attemptID,
			PsuID:       uuid.New(),
			ConnectorID: s.connectorID,
		}, nil
	})
	s.env.OnActivity(activities.StoragePSUBankBridgeConnectionAttemptsUpdateStatusActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req SendEvents) error {
		s.NotNil(req.UserLinkStatus)
		s.Equal(attemptID, req.UserLinkStatus.AttemptID)
		return nil
	})

	// Mock for Account (default case)
	s.env.OnWorkflow(RunStoreWebhookTranslation, mock.Anything, mock.Anything).Once().Return(func(ctx workflow.Context, req StoreWebhookTranslation) error {
		s.Equal(s.connectorID, req.ConnectorID)
		s.NotNil(req.Account)
		s.Equal(s.accountID.Reference, req.Account.Reference)
		return nil
	})

	s.env.ExecuteWorkflow(RunHandleWebhooks, HandleWebhooks{
		ConnectorID: s.connectorID,
		URLPath:     "/test",
		Webhook: models.Webhook{
			ID:          "test",
			ConnectorID: s.connectorID,
			QueryValues: map[string][]string{
				"test": {"test"},
			},
			Headers: map[string][]string{
				"test": {"test"},
			},
			Body: []byte(`{}`),
		},
		Config: &models.WebhookConfig{
			Name:        "test",
			ConnectorID: s.connectorID,
			URLPath:     "/test",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}
