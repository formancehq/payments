package workflow

import (
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

// connectorPollingPeriod returns the connector's polling period.
//
// For new workflow executions it reads the value deterministically via a storage-backed
// activity, so the polling period that flows into TemporalScheduleCreate is recorded in
// history. This avoids a non-determinism panic if the connector's polling period is changed
// (via UpdateConnector, which does not terminate running workflows) while a workflow that
// already scheduled is later replayed (EN-1093 / H12).
//
// The legacy (DefaultVersion) branch reads the in-process connectors manager and is retained
// for in-flight workflows until the next major version. It cannot be exercised via Temporal's
// TestWorkflowEnvironment (which always reports the newest GetVersion), so only the activity
// branch is unit-tested.
func (w Workflow) connectorPollingPeriod(ctx workflow.Context, connectorID models.ConnectorID) (time.Duration, error) {
	if IsDeterministicPollingPeriodEnabled(ctx) {
		pollingPeriod, err := activities.StorageConnectorsGetPollingPeriod(infiniteRetryContext(ctx), connectorID)
		if err != nil {
			return 0, fmt.Errorf("getting connector polling period: %w", err)
		}
		return pollingPeriod, nil
	}

	config, err := w.connectors.GetConfig(connectorID)
	if err != nil {
		return 0, err
	}
	return config.PollingPeriod, nil
}
