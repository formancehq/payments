package workflow

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/plugins/public/dummypay"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
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
	expectedScheduleID := s.w.bootstrapScheduleID(connectorID)

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

	// The new bootstrap branch replaces the ExecuteChildWorkflow(RunBootstrapTasks, ...)
	// call with StorageSchedulesStore + TemporalScheduleCreate activities —
	// the schedule triggers RunBootstrapTasks out-of-band, which the
	// unit-test env cannot observe.
	s.env.OnActivity(activities.StorageSchedulesStoreActivity, mock.Anything, mock.Anything).Once().Return(
		func(ctx context.Context, schedule models.Schedule) error {
			s.Equal(expectedScheduleID, schedule.ID)
			s.Equal(connectorID, schedule.ConnectorID)
			return nil
		},
	)
	s.env.OnActivity(activities.TemporalScheduleCreateActivity, mock.Anything, mock.Anything).Once().Return(
		func(ctx context.Context, opts activities.ScheduleCreateOptions) error {
			s.Equal(expectedScheduleID, opts.ScheduleID)
			s.Equal(RunBootstrapTasks, opts.Action.Workflow)
			s.True(opts.TriggerImmediately)
			s.Equal(expectedScheduleID, opts.SearchAttributes[SearchAttributeScheduleID])
			s.Equal(s.w.stack, opts.SearchAttributes[SearchAttributeStack])
			return nil
		},
	)
	s.env.OnWorkflow(RunScheduleConnectorHealthCheck, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunInstallConnector, InstallConnector{
		ConnectorID: connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_InstallConnector_WithBootstrap_OnFailure_DeletesBootstrapSchedule() {
	connectorID := s.installBootstrapCapableConnector()
	expectedScheduleID := s.w.bootstrapScheduleID(connectorID)

	// Force the install to fail after the plugin is loaded but before the
	// bootstrap schedule is created — runInstallConnector must still invoke
	// TemporalScheduleDelete + StorageSchedulesDelete because the plugin
	// declares BootstrapOnInstall, so the failure branch cannot know whether
	// the schedule was created.
	s.env.OnActivity(activities.PluginInstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(
		nil, errors.New("install failed"),
	)
	s.env.OnActivity(activities.StorageConnectorsDeleteActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, mock.Anything).Once().Return(
		func(ctx context.Context, scheduleID string) error {
			s.Equal(expectedScheduleID, scheduleID)
			return nil
		},
	)
	s.env.OnActivity(activities.StorageSchedulesDeleteActivity, mock.Anything, mock.Anything).Once().Return(
		func(ctx context.Context, scheduleID string) error {
			s.Equal(expectedScheduleID, scheduleID)
			return nil
		},
	)

	s.env.ExecuteWorkflow(RunInstallConnector, InstallConnector{
		ConnectorID: connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.Error(s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_InstallConnector_WithBootstrap_CleanupDeleteFailure_DoesNotMaskInstallError() {
	// Regression guard for the highest-risk leak class: install fails, cleanup
	// is invoked, and one or both cleanup deletes themselves fail. The
	// workflow must still surface the primary install error (not a cleanup
	// error), and `workflow.GetLogger(ctx).Error(...)` must absorb the
	// cleanup failures per the log-and-swallow contract at
	// install_connector.go:53-58.
	//
	// All mocked errors use NewNonRetryableApplicationError so the activity
	// retry policy (see context.go: NonRetryableErrorTypes is empty) does not
	// treat them as retryable. This mirrors a terminal failure semantic for
	// this regression test and keeps the mock counts deterministic.
	connectorID := s.installBootstrapCapableConnector()
	expectedScheduleID := s.w.bootstrapScheduleID(connectorID)

	s.env.OnActivity(activities.PluginInstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(
		nil, temporal.NewNonRetryableApplicationError("primary install error", "installFailed", nil),
	)
	s.env.OnActivity(activities.StorageConnectorsDeleteActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.TemporalScheduleDeleteActivity, mock.Anything, mock.Anything).Once().Return(
		func(ctx context.Context, scheduleID string) error {
			s.Equal(expectedScheduleID, scheduleID)
			return temporal.NewNonRetryableApplicationError("temporal schedule delete failed", "cleanupFailed", nil)
		},
	)
	s.env.OnActivity(activities.StorageSchedulesDeleteActivity, mock.Anything, mock.Anything).Once().Return(
		func(ctx context.Context, scheduleID string) error {
			s.Equal(expectedScheduleID, scheduleID)
			return temporal.NewNonRetryableApplicationError("storage schedule delete failed", "cleanupFailed", nil)
		},
	)

	s.env.ExecuteWorkflow(RunInstallConnector, InstallConnector{
		ConnectorID: connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
	// The workflow error must carry the primary install error, NOT the
	// cleanup-delete error messages — log-and-swallow contract.
	s.Contains(err.Error(), "primary install error")
	s.NotContains(err.Error(), "temporal schedule delete failed")
	s.NotContains(err.Error(), "storage schedule delete failed")
}

func (s *UnitTestSuite) Test_InstallConnector_WithEmptyBootstrap_OnFailure_DoesNotDeleteSchedule() {
	connectorID := s.installEmptyBootstrapConnector()

	// No bootstrap schedule was ever created (empty BootstrapOnInstall), so
	// the failure branch must NOT invoke the schedule-delete activities.
	s.env.OnActivity(activities.PluginInstallConnectorActivity, mock.Anything, mock.Anything).Once().Return(
		nil, errors.New("install failed"),
	)
	s.env.OnActivity(activities.StorageConnectorsDeleteActivity, mock.Anything, mock.Anything).Once().Return(nil)
	// TemporalScheduleDeleteActivity + StorageSchedulesDeleteActivity must not be called.

	s.env.ExecuteWorkflow(RunInstallConnector, InstallConnector{
		ConnectorID: connectorID,
	})

	s.True(s.env.IsWorkflowCompleted())
	s.Error(s.env.GetWorkflowError())
}

