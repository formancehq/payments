package workflow

import (
	"context"
	"errors"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func (s *UnitTestSuite) Test_HandleWebhooks_Success() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return(
		[]models.WebhookConfig{
			{
				Name:        "test",
				ConnectorID: s.connectorID,
				URLPath:     "/test",
			},
		},
		nil,
	)
	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
		return &models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{
				{
					IdempotencyKey:  "test",
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
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_HandleWebhooks_NoResponses_Success() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return(
		[]models.WebhookConfig{
			{
				Name:        "test",
				ConnectorID: s.connectorID,
				URLPath:     "/test",
			},
		},
		nil,
	)
	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
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
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_HandleWebhooks_NoConfigForWebhook_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return(
		[]models.WebhookConfig{},
		nil,
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
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	s.ErrorContains(err, "webhook config not found")
}

func (s *UnitTestSuite) Test_HandleWebhooks_StorageWebhooksConfigsGet_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "test", errors.New("test")),
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
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_HandleWebhooks_StorageWebhooksStore_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return(
		[]models.WebhookConfig{
			{
				Name:        "test",
				ConnectorID: s.connectorID,
				URLPath:     "/test",
			},
		},
		nil,
	)
	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "test", errors.New("test")),
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
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_HandleWebhooks_PluginTranslateWebhook_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return(
		[]models.WebhookConfig{
			{
				Name:        "test",
				ConnectorID: s.connectorID,
				URLPath:     "/test",
			},
		},
		nil,
	)
	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(nil,
		temporal.NewNonRetryableApplicationError("test", "test", errors.New("test")),
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
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_HandleWebhooks_RunStoreWebhookTranslation_Error() {
	s.env.OnActivity(activities.StorageWebhooksConfigsGetActivity, mock.Anything, s.connectorID).Once().Return(
		[]models.WebhookConfig{
			{
				Name:        "test",
				ConnectorID: s.connectorID,
				URLPath:     "/test",
			},
		},
		nil,
	)
	s.env.OnActivity(activities.StorageWebhooksStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.PluginTranslateWebhookActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.TranslateWebhookRequest) (*models.TranslateWebhookResponse, error) {
		return &models.TranslateWebhookResponse{
			Responses: []models.WebhookResponse{
				{
					IdempotencyKey:  "test",
					Account:         &s.pspAccount,
					ExternalAccount: &s.pspAccount,
					Payment:         &s.pspPayment,
				},
			},
		}, nil
	})
	s.env.OnWorkflow(RunStoreWebhookTranslation, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "test", errors.New("test")),
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
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}
