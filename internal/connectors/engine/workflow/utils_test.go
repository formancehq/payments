package workflow

import (
	"fmt"
	"testing"

	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/google/uuid"
)

// payoutTaskQueue mirrors engine.GetPayoutTaskQueue for tests. The workflow
// package cannot import engine (engine imports workflow), so the format is
// duplicated here and pinned by TestTaskQueueFormats below.
func payoutTaskQueue(stack string, connectorID models.ConnectorID) string {
	return fmt.Sprintf("%s-%s-payout", stack, connectorID.String())
}

// The payout-throttle routing relies on this package and the engine package
// producing byte-identical task queue names: a payout workflow re-routes its
// non-PSP activities to getDefaultTaskQueue(), and the worker listening there
// was created under engine.GetDefaultTaskQueue(). Both formats are separate
// fmt.Sprintf calls in packages that cannot import each other, so pin them from
// both sides against a literal. The matching assertions live in
// internal/connectors/engine/engine_test.go.
func TestTaskQueueFormats(t *testing.T) {
	w := Workflow{stack: "somestack"}
	if got, want := w.getDefaultTaskQueue(), "somestack-default"; got != want {
		t.Fatalf("getDefaultTaskQueue() = %q, want %q (must match engine.GetDefaultTaskQueue)", got, want)
	}

	connectorID := models.ConnectorID{Reference: uuid.New(), Provider: "someprovider"}
	want := "somestack-" + connectorID.String() + "-payout"
	if got := payoutTaskQueue("somestack", connectorID); got != want {
		t.Fatalf("payoutTaskQueue() = %q, want %q (must match engine.GetPayoutTaskQueue)", got, want)
	}
}
