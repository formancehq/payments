package workflow

import (
	"context"
	"errors"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_CreateWebhooks_Success() {
	s.env.OnActivity(activities.PluginCreateWebhooksActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CreateWebhooksRequest) (*models.CreateWebhooksResponse, error) {
		s.Equal(s.connectorID, req.ConnectorID)
		s.Equal(s.connectorID.String(), req.Req.ConnectorID)
		s.Equal("http://localhost:8080/api/payments/v3/connectors/webhooks/"+s.connectorID.String(), req.Req.WebhookBaseUrl)
		return &models.CreateWebhooksResponse{
			Others: []models.PSPOther{
				{
					ID:    "test",
					Other: []byte(`{}`),
				},
			},
		}, nil
	})
	s.env.OnWorkflow(Run, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunCreateWebhooks, CreateWebhooks{
		ConnectorID: s.connectorID,
		Config:      models.DefaultConfig(),
		FromPayload: nil,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_CreateWebhooks_PluginCreateWebhooksActivity_Error() {
	s.env.OnActivity(activities.PluginCreateWebhooksActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "PLUGIN", errors.New("test")),
	)

	s.env.ExecuteWorkflow(RunCreateWebhooks, CreateWebhooks{
		ConnectorID: s.connectorID,
		Config:      models.DefaultConfig(),
		FromPayload: nil,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_CreateWebhooks_Run_Error() {
	s.env.OnActivity(activities.PluginCreateWebhooksActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.CreateWebhooksRequest) (*models.CreateWebhooksResponse, error) {
		s.Equal(s.connectorID, req.ConnectorID)
		s.Equal(s.connectorID.String(), req.Req.ConnectorID)
		s.Equal("http://localhost:8080/api/payments/v3/connectors/webhooks/"+s.connectorID.String(), req.Req.WebhookBaseUrl)
		return &models.CreateWebhooksResponse{
			Others: []models.PSPOther{
				{
					ID:    "test",
					Other: []byte(`{}`),
				},
			},
		}, nil
	})
	s.env.OnWorkflow(Run, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "WORKFLOW", errors.New("test")),
	)

	s.env.ExecuteWorkflow(RunCreateWebhooks, CreateWebhooks{
		ConnectorID: s.connectorID,
		Config:      models.DefaultConfig(),
		FromPayload: nil,
	}, []models.ConnectorTaskTree{})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}
