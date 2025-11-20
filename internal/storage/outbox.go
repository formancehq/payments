package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	internalTime "github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	internalErrors "github.com/formancehq/payments/internal/utils/errors"
	"github.com/uptrace/bun"
)

type outboxEvent struct {
	bun.BaseModel `bun:"table:outbox_events"`

	// Primary key
	ID models.EventID `bun:"id,pk,type:character varying,notnull"`

	// Mandatory fields
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

	// Filter out events that already exist in events_sent table
	toInsert := make([]outboxEvent, 0, len(events))
	for _, event := range events {
		// EventID must be set on the event
		if event.ID.EventIdempotencyKey == "" {
			return e("event ID must be set with EventIdempotencyKey", errors.New("missing event ID"))
		}

		// Check if this EventID already exists in events_sent using the transaction
		exists, err := tx.NewSelect().
			Model((*eventSent)(nil)).
			Where("id = ?", event.ID).
			Exists(ctx)
		if err != nil {
			return e("failed to check if event already sent", err)
		}

		// Skip events that already exist in events_sent
		if exists {
			continue
		}

		toInsert = append(toInsert, fromOutboxEventModel(event))
	}

	// If all events were filtered out, nothing to insert
	if len(toInsert) == 0 {
		return nil
	}

	// Insert with ON CONFLICT DO NOTHING for (id)
	_, err := tx.NewInsert().
		Model(&toInsert).
		On("CONFLICT (id) DO NOTHING").
		Exec(ctx)

	return e("failed to insert outbox events", err)
}

// OutboxEventsInsertWithTx is meant to be used only when we don't have a related entity;
// outbox event should always be inserted at the same time as the entity they're linked to.
func (s *store) OutboxEventsInsertWithTx(ctx context.Context, events []models.OutboxEvent) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return e("failed to begin transaction", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	if err = s.OutboxEventsInsert(ctx, tx, events); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return e("failed to commit transaction", err)
	}
	return nil
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

func (s *store) OutboxEventsMarkFailed(ctx context.Context, eventID models.EventID, retryCount int, err error) error {
	now := internalTime.Now().UTC()
	maxRetries := models.MaxOutboxRetries

	// Determine status based on retry count or error type
	status := models.OUTBOX_STATUS_PENDING
	var nonRetriable internalErrors.NonRetryableError
	if retryCount >= maxRetries || errors.As(err, &nonRetriable) {
		status = models.OUTBOX_STATUS_FAILED
	}
	var errMsg *string
	if err != nil {
		msg := err.Error()
		errMsg = &msg
	}

	_, updateErr := s.db.NewUpdate().
		TableExpr("outbox_events").
		Set("status = ?", status).
		Set("retry_count = ?", retryCount).
		Set("last_retry_at = ?", now).
		Set("error = ?", errMsg).
		Where("id = ?", eventID).
		Exec(ctx)

	return e("failed to mark outbox event", updateErr)
}

func (s *store) OutboxEventsMarkProcessedAndRecordSent(ctx context.Context, eventIDs []models.EventID, eventsSent []models.EventSent) error {
	if len(eventIDs) == 0 {
		return nil // Nothing to do
	}

	if len(eventIDs) != len(eventsSent) {
		return e("eventIDs and eventsSent must have the same length", errors.New("length mismatch"))
	}

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return e("failed to create transaction", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	// Mark events as processed (batch update using id)
	_, err = tx.NewUpdate().
		TableExpr("outbox_events").
		Set("status = ?", models.OUTBOX_STATUS_PROCESSED).
		Where("id IN (?)", bun.In(eventIDs)).
		Exec(ctx)
	if err != nil {
		return e("failed to mark published events as processed", err)
	}

	// Record in events_sent table (batch insert)
	toInsert := make([]eventSent, 0, len(eventsSent))
	for _, eventSent := range eventsSent {
		toInsert = append(toInsert, fromEventSentModel(eventSent))
	}

	if len(toInsert) > 0 {
		_, err = tx.NewInsert().
			Model(&toInsert).
			On("CONFLICT (id) DO NOTHING").
			Exec(ctx)
		if err != nil {
			return e("failed to record events as sent", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return e("failed to commit transaction", err)
	}

	return nil
}

func fromOutboxEventModel(from models.OutboxEvent) outboxEvent {
	return outboxEvent{
		ID:          from.ID,
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
}

func toOutboxEventModel(from outboxEvent) models.OutboxEvent {
	return models.OutboxEvent{
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
