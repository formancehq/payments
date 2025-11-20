package activities

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

func (a Activities) OutboxDeleteOldProcessedEvents(ctx context.Context) error {
	// Calculate cutoff date: 1 month ago
	cutoffDate := time.Now().UTC().AddDate(0, -1, 0)

	// Delete old processed events
	if err := a.storage.OutboxEventsDeleteOldProcessed(ctx, cutoffDate); err != nil {
		return fmt.Errorf("failed to delete old processed outbox events: %w", err)
	}

	return nil
}

var OutboxDeleteOldProcessedEventsActivity = Activities{}.OutboxDeleteOldProcessedEvents

func OutboxDeleteOldProcessedEvents(ctx workflow.Context) error {
	return executeActivity(ctx, OutboxDeleteOldProcessedEventsActivity, nil)
}
