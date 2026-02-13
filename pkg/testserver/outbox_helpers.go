package testserver

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/onsi/gomega"
	"github.com/uptrace/bun"
)

// AwaitOutboxEvent polls the database outbox_events table until it finds
// an event with the given outbox event type matching all provided payload matchers,
// or the timeout expires. It checks outbox rows regardless of status to avoid
// races with the outbox publisher moving events from pending to processed.
func AwaitOutboxEvent(ctx context.Context, s *Server, outboxEventType string, timeout, interval time.Duration, matchers ...PayloadMatcher) error {
	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return errors.New("timeout waiting for outbox event of type " + outboxEventType)
		}

		found, err := findMatchingOutboxEvent(ctx, s, outboxEventType, false, matchers...)
		if err != nil {
			return err
		}
		if found {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

// EventuallyOutbox is a convenience wrapper that uses default polling
// timings aligned with the outbox publisher period.
func EventuallyOutbox(ctx context.Context, s *Server, outboxEventType string, matchers ...PayloadMatcher) error {
	return AwaitOutboxEvent(ctx, s, outboxEventType, 5*time.Second, 50*time.Millisecond, matchers...)
}

// findMatchingOutboxEvent queries the DB for outbox events of the given type
// and applies matchers to the payload.
func findMatchingOutboxEvent(ctx context.Context, s *Server, outboxEventType string, onlyPending bool, matchers ...PayloadMatcher) (bool, error) {
	db, err := s.Database()
	if err != nil {
		return false, err
	}
	defer func(db *bun.DB) { _ = db.Close() }(db)

	type outboxRow struct {
		Payload   json.RawMessage `bun:"payload"`
		CreatedAt time.Time       `bun:"created_at"`
	}

	var rows []outboxRow
	query := db.NewSelect().
		TableExpr("outbox_events").
		Where("event_type = ?", outboxEventType)
	if onlyPending {
		query = query.Where("status = ?", "pending")
	}
	err = query.
		Order("created_at DESC").
		Limit(200).
		Scan(ctx, &rows)
	if err != nil {
		return false, err
	}

	for _, row := range rows {
		matched := true
		for _, m := range matchers {
			if err := m.Match(row.Payload); err != nil {
				matched = false
				break
			}
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

// MustEventuallyOutbox is a gomega-friendly helper for tests using Gomega directly.
func MustEventuallyOutbox(ctx context.Context, s *Server, outboxEventType string, matchers ...PayloadMatcher) {
	gomega.Expect(EventuallyOutbox(ctx, s, outboxEventType, matchers...)).To(gomega.Succeed())
}

func MustOutbox(ctx context.Context, s *Server, outboxEventType string, matchers ...PayloadMatcher) {
	success, err := findMatchingOutboxEvent(ctx, s, outboxEventType, true, matchers...)
	gomega.Expect(err).To(gomega.BeNil())
	gomega.Expect(success).To(gomega.BeTrue())
}

// CountOutboxEventsByType counts events with the given type in outbox_events table.
// Test helper to standardize counting across E2E tests.
func CountOutboxEventsByType(ctx context.Context, s *Server, eventType string) (int, error) {
	db, err := s.Database()
	if err != nil {
		return 0, err
	}
	defer func(db *bun.DB) { _ = db.Close() }(db)

	type row struct {
		Count int `bun:"count"`
	}
	var r row
	err = db.NewSelect().
		TableExpr("outbox_events").
		ColumnExpr("count(*) as count").
		Where("event_type = ?", eventType).
		Scan(ctx, &r)
	return r.Count, err
}

// LoadOutboxPayloadsByType returns payloads for events of the given type ordered by creation time.
func LoadOutboxPayloadsByType(ctx context.Context, s *Server, eventType string) ([]json.RawMessage, error) {
	db, err := s.Database()
	if err != nil {
		return nil, err
	}
	defer func(db *bun.DB) { _ = db.Close() }(db)

	type row struct {
		Payload json.RawMessage `bun:"payload"`
	}
	var rows []row
	err = db.NewSelect().
		TableExpr("outbox_events").
		Column("payload").
		Where("event_type = ?", eventType).
		Order("created_at ASC").
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	out := make([]json.RawMessage, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Payload)
	}
	return out, nil
}
