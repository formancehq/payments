package activities

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/pkg/domain/models"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// StorageConnectorsGetPollingPeriod returns only the connector's polling period — a
// non-secret value read deterministically (recorded in workflow history) so that schedule
// creation does not depend on mutable in-process manager state, and so a polling-period
// change on replay cannot make the workflow non-deterministic (EN-1093 / H12). It MUST
// never return the connector config or any secret: prod Temporal payload encryption is OFF,
// so activity results are stored in clear in workflow history.
func (a Activities) StorageConnectorsGetPollingPeriod(ctx context.Context, connectorID models.ConnectorID) (time.Duration, error) {
	connector, err := a.storage.ConnectorsGet(ctx, connectorID)
	if err != nil {
		return 0, temporalStorageError(err)
	}

	// Parse onto the configurer's default config (exactly like manager.Load / GetConfig) so
	// an absent pollingPeriod resolves to the default rather than zero — otherwise migrated
	// configs (which may omit it) would yield a zero-interval schedule.
	cfg := a.connectors.DefaultConfig()
	if len(connector.Config) > 0 {
		if err := json.Unmarshal(connector.Config, &cfg); err != nil {
			// Corrupt config will not self-heal; fail fast instead of retrying forever.
			return 0, temporal.NewNonRetryableApplicationError("invalid connector config", ErrTypeInvalidArgument, err)
		}
	}

	return cfg.PollingPeriod, nil
}

var StorageConnectorsGetPollingPeriodActivity = Activities{}.StorageConnectorsGetPollingPeriod

func StorageConnectorsGetPollingPeriod(ctx workflow.Context, connectorID models.ConnectorID) (time.Duration, error) {
	var pollingPeriod time.Duration
	if err := executeActivity(ctx, StorageConnectorsGetPollingPeriodActivity, &pollingPeriod, connectorID); err != nil {
		return 0, err
	}
	return pollingPeriod, nil
}
