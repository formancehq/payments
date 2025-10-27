package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	internalTime "github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type outboxEvent struct {
	bun.BaseModel `bun:"table:outbox_events"`

	// Autoincrement fields
	SortID int64 `bun:"sort_id,autoincrement"`

	// Mandatory fields
	ID        uuid.UUID                `bun:"id,pk,type:uuid"`
	EventType string                   `bun:"event_type,type:text,notnull"`
	EntityID  string                   `bun:"entity_id,type:character varying,notnull"`
	Payload   json.RawMessage          `bun:"payload,type:jsonb,notnull"`
	CreatedAt internalTime.Time        `bun:"created_at,type:timestamp without time zone,notnull"`
	Status    models.OutboxEventStatus `bun:"status,type:text,notnull"`

	// Optional fields
	ConnectorID *models.ConnectorID `bun:"connector_id,type:character varying,nullzero"`
	RetryCount  int                 `bun:"retry_count,type:integer,notnull"`
	LastRetryAt *internalTime.Time  `bun:"last_retry_at,type:timestamp without time zone,nullzero"`
	Error       *string             `bun:"error,type:text,nullzero"`
}

func (s *store) OutboxEventsInsert(ctx context.Context, tx bun.Tx, events []models.OutboxEvent) error {
	if len(events) == 0 {
		return nil
	}

	toInsert := make([]outboxEvent, 0, len(events))
	for _, event := range events {
		toInsert = append(toInsert, fromOutboxEventModel(event))
	}

	_, err := tx.NewInsert().
		Model(&toInsert).
		Exec(ctx)

	return e("failed to insert outbox events", err)
}

func (s *store) OutboxEventsPollPending(ctx context.Context, limit int) ([]models.OutboxEvent, error) {
	var events []outboxEvent

	err := s.db.NewSelect().
		TableExpr("outbox_events").
		Where("status = ?", models.OUTBOX_STATUS_PENDING).
		Order("created_at ASC").
		Limit(limit).
		Scan(ctx, &events)

	if err != nil {
		return nil, e("failed to poll pending outbox events", err)
	}

	result := make([]models.OutboxEvent, 0, len(events))
	for _, event := range events {
		result = append(result, toOutboxEventModel(event))
	}

	return result, nil
}

func (s *store) OutboxEventsMarkFailed(ctx context.Context, id uuid.UUID, retryCount int, errorMsg string) error {
	now := internalTime.Now().UTC()
	maxRetries := models.MaxOutboxRetries

	// Determine status based on retry count
	status := models.OUTBOX_STATUS_PENDING
	if retryCount >= maxRetries {
		status = models.OUTBOX_STATUS_FAILED
	}

	_, err := s.db.NewUpdate().
		TableExpr("outbox_events").
		Set("status = ?", status).
		Set("retry_count = ?", retryCount).
		Set("last_retry_at = ?", now).
		Set("error = ?", errorMsg).
		Where("id = ?", id).
		Exec(ctx)

	return e("failed to mark outbox event", err)
}

func (s *store) OutboxEventsDeleteAndRecordSent(ctx context.Context, eventID uuid.UUID, eventSent models.EventSent) error {
	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Delete from outbox
	_, err = tx.NewDelete().
		TableExpr("outbox_events").
		Where("id = ?", eventID).
		Exec(ctx)
	if err != nil {
		return e("failed to delete published event from outbox", err)
	}

	// Record in events_sent table
	toInsert := fromEventSentModel(eventSent)
	_, err = tx.NewInsert().
		Model(&toInsert).
		On("CONFLICT (id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return e("failed to record event as sent", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func fromOutboxEventModel(from models.OutboxEvent) outboxEvent {
	event := outboxEvent{
		EventType:   from.EventType,
		EntityID:    from.EntityID,
		Payload:     from.Payload,
		CreatedAt:   internalTime.New(from.CreatedAt),
		Status:      from.Status,
		ConnectorID: from.ConnectorID,
		RetryCount:  from.RetryCount,
		LastRetryAt: func() *internalTime.Time {
			if from.LastRetryAt == nil {
				return nil
			}
			return pointer.For(internalTime.New(*from.LastRetryAt))
		}(),
		Error: from.Error,
	}
	// Generate UUID if not provided
	if from.ID == uuid.Nil {
		event.ID = uuid.New()
	} else {
		event.ID = from.ID
	}
	// SortID is auto-increment, don't set it manually unless provided
	if from.SortID != 0 {
		event.SortID = from.SortID
	}
	return event
}

func toOutboxEventModel(from outboxEvent) models.OutboxEvent {
	return models.OutboxEvent{
		SortID:      from.SortID,
		ID:          from.ID,
		EventType:   from.EventType,
		EntityID:    from.EntityID,
		Payload:     from.Payload,
		CreatedAt:   from.CreatedAt.Time,
		Status:      from.Status,
		ConnectorID: from.ConnectorID,
		RetryCount:  from.RetryCount,
		LastRetryAt: func() *time.Time {
			if from.LastRetryAt == nil {
				return nil
			}
			return pointer.For(from.LastRetryAt.Time)
		}(),
		Error: from.Error,
	}
}
