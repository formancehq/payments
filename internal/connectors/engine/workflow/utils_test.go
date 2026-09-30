package workflow

import (
	"errors"
	"fmt"
	"testing"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
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

// The whole point of the NOT_INITIATED status is that consumers can tell a PSP
// that never took the payment on from one that took it and then failed. That
// distinction is carried by the temporal error type, which is all that survives
// the activity boundary, so pin every classification it can arrive with.
func TestPIFailureStatus(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want models.PaymentInitiationAdjustmentStatus
	}{
		{
			name: "nil",
			err:  nil,
			want: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
		},
		{
			name: "plain error",
			err:  errors.New("boom"),
			want: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
		},
		{
			name: "retryable default",
			err:  temporal.NewApplicationErrorWithCause("boom", activities.ErrTypeDefault, errors.New("boom")),
			want: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
		},
		{
			name: "unimplemented",
			err:  temporal.NewNonRetryableApplicationError("boom", activities.ErrTypeUnimplemented, errors.New("boom")),
			want: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_NOT_INITIATED,
		},
		{
			name: "invalid argument",
			err:  temporal.NewNonRetryableApplicationError("boom", activities.ErrTypeInvalidArgument, errors.New("boom")),
			want: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_NOT_INITIATED,
		},
		{
			name: "wrapped invalid argument",
			err:  fmt.Errorf("activity error: %w", temporal.NewNonRetryableApplicationError("boom", activities.ErrTypeInvalidArgument, errors.New("boom"))),
			want: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_NOT_INITIATED,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := piFailureStatus(tc.err); got != tc.want {
				t.Fatalf("piFailureStatus() = %q, want %q", got, tc.want)
			}
		})
	}
}
