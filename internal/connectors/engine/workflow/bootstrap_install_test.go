package workflow

import (
	"context"
	"encoding/json"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/plugins/public/dummypay"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// bootstrapCapablePlugin wraps dummypay.Plugin with a BootstrapOnInstall
// declaration so install_connector takes the new detached-bootstrap branch.
type bootstrapCapablePlugin struct {
	*dummypay.Plugin
}

func (bootstrapCapablePlugin) BootstrapOnInstall() []models.TaskType {
	return []models.TaskType{models.TASK_FETCH_ACCOUNTS}
}

func (s *UnitTestSuite) installBootstrapCapableConnector() models.ConnectorID {
	const provider = "test-bootstrap"
	registry.RegisterPlugin(provider, models.PluginTypePSP, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		p, err := dummypay.New(name, logger, rm)
		if err != nil {
			return nil, err
		}
		return bootstrapCapablePlugin{Plugin: p}, nil
	}, []models.Capability{}, struct{}{}, 25)

	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  provider,
	}
	connector := models.Connector{
		ConnectorBase: models.ConnectorBase{
			ID:        connectorID,
			Name:      "bootstrap-test",
			CreatedAt: s.env.Now().UTC(),
			Provider:  provider,
		},
		Config: []byte(`{"name":"bootstrap-test","directory":"/tmp"}`),
	}
	_, _, err := s.w.connectors.Load(connector, true, true)
	s.NoError(err)
	return connectorID
}

// emptyBootstrapPlugin implements PluginWithBootstrapOnInstall but returns
// an empty slice — the install workflow should still take the legacy
// RunNextTasksV3_1 path, not the detached bootstrap path.
type emptyBootstrapPlugin struct {
	*dummypay.Plugin
}

func (emptyBootstrapPlugin) BootstrapOnInstall() []models.TaskType {
	return nil
}

func (s *UnitTestSuite) installEmptyBootstrapConnector() models.ConnectorID {
	const provider = "test-empty-bootstrap"
	registry.RegisterPlugin(provider, models.PluginTypePSP, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		p, err := dummypay.New(name, logger, rm)
		if err != nil {
			return nil, err
		}
		return emptyBootstrapPlugin{Plugin: p}, nil
	}, []models.Capability{}, struct{}{}, 25)

	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  provider,
	}
	connector := models.Connector{
		ConnectorBase: models.ConnectorBase{
			ID:        connectorID,
			Name:      "empty-bootstrap-test",
			CreatedAt: s.env.Now().UTC(),
			Provider:  provider,
		},
		Config: []byte(`{"name":"empty-bootstrap-test","directory":"/tmp"}`),
	}
	_, _, err := s.w.connectors.Load(connector, true, true)
	s.NoError(err)
	return connectorID
}

func (s *UnitTestSuite) Test_InstallConnector_WithEmptyBootstrap_UsesLegacyPath() {
	connectorID := s.installEmptyBootstrapConnector()

	s.env.OnActivity(activities.PluginInstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(
		func(ctx context.Context, req activities.InstallConnectorRequest) (*models.InstallResponse, error) {
			return &models.InstallResponse{Workflow: []models.ConnectorTaskTree{}}, nil
		},
	)
	s.env.OnActivity(activities.StorageConnectorTasksTreeStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	// Empty BootstrapOnInstall → fallthrough to RunNextTasksV3_1.
	s.env.OnWorkflow(RunNextTasksV3_1, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunScheduleConnectorHealthCheck, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunInstallConnector, InstallConnector{
		ConnectorID: connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_InstallConnector_WithBootstrap_StartsBootstrapWorkflow() {
	connectorID := s.installBootstrapCapableConnector()

	s.env.OnActivity(activities.PluginInstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(
		func(ctx context.Context, req activities.InstallConnectorRequest) (*models.InstallResponse, error) {
			return &models.InstallResponse{
				Workflow: []models.ConnectorTaskTree{
					{TaskType: models.TASK_FETCH_ACCOUNTS},
				},
			}, nil
		},
	)
	s.env.OnActivity(activities.StorageConnectorTasksTreeStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	// The new bootstrap branch starts RunBootstrapTasks (detached) and
	// skips the direct RunNextTasksV3_1 launch — that is now owned by
	// RunBootstrapTasks itself and fires after bootstrap completion.
	s.env.OnWorkflow(RunBootstrapTasks, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunScheduleConnectorHealthCheck, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunInstallConnector, InstallConnector{
		ConnectorID: connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

