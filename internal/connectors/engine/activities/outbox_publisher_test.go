package activities_test

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("OutboxPublishPendingEvents", func() {
	var (
		act           activities.Activities
		s             *storage.MockStorage
		evts          *events.Events
		mockPublisher *activities.MockPublisher
		logger        = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		s = storage.NewMockStorage(ctrl)
		mockPublisher = activities.NewMockPublisher(ctrl)
		evts = events.New(mockPublisher, "http://localhost")
		act = activities.New(logger, nil, s, evts, nil, 0)
	})

	Context("when polling pending outbox events", func() {
		It("successfully publishes events", func(ctx SpecContext) {
			connectorID := &models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.New(),
			}

			testEvents := []models.OutboxEvent{
				{
					ID:             uuid.New(),
					EventType:      "account.saved",
					EntityID:       "acc_123",
					Payload:        json.RawMessage(`{"id":"acc_123","name":"Test Account"}`),
					CreatedAt:      time.Now().UTC(),
					Status:         models.OUTBOX_STATUS_PENDING,
					ConnectorID:    connectorID,
					RetryCount:     0,
					IdempotencyKey: "account.saved:acc_123",
				},
			}

			// Poll events
			s.EXPECT().
				OutboxEventsPollPending(ctx, 100).
				Return(testEvents, nil)

			// Publish event (expect generic topic from events.Publish)
			mockPublisher.EXPECT().
				Publish(gomock.Any(), gomock.Any()).
				Return(nil)

			// Delete from outbox and record sent atomically
			s.EXPECT().
				OutboxEventsDeleteAndRecordSent(ctx, testEvents[0].ID, gomock.Any()).
				Do(func(_ context.Context, eventID uuid.UUID, eventSent models.EventSent) {
					Expect(eventSent.ConnectorID).To(Equal(connectorID))
					Expect(eventSent.ID.EventIdempotencyKey).To(Equal("account.saved:acc_123"))
				}).
				Return(nil)

			err := act.OutboxPublishPendingEvents(ctx, 100)
			Expect(err).To(BeNil())
		})

		It("returns nil when no events are pending", func(ctx SpecContext) {
			s.EXPECT().
				OutboxEventsPollPending(ctx, 100).
				Return([]models.OutboxEvent{}, nil)

			err := act.OutboxPublishPendingEvents(ctx, 100)
			Expect(err).To(BeNil())
		})

		It("handles poll error", func(ctx SpecContext) {
			expectedErr := errors.New("database error")
			s.EXPECT().
				OutboxEventsPollPending(ctx, 100).
				Return(nil, expectedErr)

			err := act.OutboxPublishPendingEvents(ctx, 100)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to poll pending outbox events"))
		})

		It("marks event as failed after max retries", func(ctx SpecContext) {
			connectorID := &models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.New(),
			}

			testEvents := []models.OutboxEvent{
				{
					ID:             uuid.New(),
					EventType:      "account.saved",
					EntityID:       "acc_123",
					Payload:        json.RawMessage(`{"id":"acc_123","name":"Test Account"}`),
					CreatedAt:      time.Now().UTC(),
					Status:         models.OUTBOX_STATUS_PENDING,
					ConnectorID:    connectorID,
					RetryCount:     models.MaxOutboxRetries, // Already at max retries
					IdempotencyKey: "account.saved:acc_123",
				},
			}

			// Poll events
			s.EXPECT().
				OutboxEventsPollPending(ctx, 100).
				Return(testEvents, nil)

			// Publish fails
			publishErr := errors.New("publish error")
			mockPublisher.EXPECT().
				Publish(gomock.Any(), gomock.Any()).
				Return(publishErr)

			// Mark as failed (exceeds max retries)
			s.EXPECT().
				OutboxEventsMarkFailed(ctx, testEvents[0].ID, models.MaxOutboxRetries+1, publishErr).
				Return(nil)

			err := act.OutboxPublishPendingEvents(ctx, 100)
			Expect(err).To(BeNil()) // Activity doesn't fail, just marks event as failed
		})

		It("retries failed event", func(ctx SpecContext) {
			connectorID := &models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.New(),
			}

			testEvents := []models.OutboxEvent{
				{
					ID:             uuid.New(),
					EventType:      "account.saved",
					EntityID:       "acc_123",
					Payload:        json.RawMessage(`{"id":"acc_123","name":"Test Account"}`),
					CreatedAt:      time.Now().UTC(),
					Status:         models.OUTBOX_STATUS_PENDING,
					ConnectorID:    connectorID,
					RetryCount:     2, // Below max retries
					IdempotencyKey: "account.saved:acc_123",
				},
			}

			// Poll events
			s.EXPECT().
				OutboxEventsPollPending(ctx, 100).
				Return(testEvents, nil)

			// Publish fails
			publishErr := errors.New("publish error")
			mockPublisher.EXPECT().
				Publish(gomock.Any(), gomock.Any()).
				Return(publishErr)

			// Mark as pending for retry
			s.EXPECT().
				OutboxEventsMarkFailed(ctx, testEvents[0].ID, 3, publishErr).
				Return(nil)

			err := act.OutboxPublishPendingEvents(ctx, 100)
			Expect(err).To(BeNil()) // Activity doesn't fail, just marks event for retry
		})

		It("handles unknown event type", func(ctx SpecContext) {
			connectorID := &models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.New(),
			}

			testEvents := []models.OutboxEvent{
				{
					ID:             uuid.New(),
					EventType:      "unknown.event",
					EntityID:       "some_id",
					Payload:        json.RawMessage(`{}`),
					CreatedAt:      time.Now().UTC(),
					Status:         models.OUTBOX_STATUS_PENDING,
					ConnectorID:    connectorID,
					RetryCount:     0,
					IdempotencyKey: "unknown.event:some_id",
				},
			}

			// Poll events
			s.EXPECT().
				OutboxEventsPollPending(ctx, 100).
				Return(testEvents, nil)

			// Mark as failed
			s.EXPECT().
				OutboxEventsMarkFailed(ctx, testEvents[0].ID, 1, gomock.Any()).
				Return(nil)

			err := act.OutboxPublishPendingEvents(ctx, 100)
			Expect(err).To(BeNil())
		})

		It("handles publish error while marking failed", func(ctx SpecContext) {
			connectorID := &models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.New(),
			}

			testEvents := []models.OutboxEvent{
				{
					ID:             uuid.New(),
					EventType:      "account.saved",
					EntityID:       "acc_123",
					Payload:        json.RawMessage(`{"id":"acc_123","name":"Test Account"}`),
					CreatedAt:      time.Now().UTC(),
					Status:         models.OUTBOX_STATUS_PENDING,
					ConnectorID:    connectorID,
					RetryCount:     0,
					IdempotencyKey: "account.saved:acc_123",
				},
			}

			// Poll events
			s.EXPECT().
				OutboxEventsPollPending(ctx, 100).
				Return(testEvents, nil)

			// Publish fails
			publishErr := errors.New("publish error")
			mockPublisher.EXPECT().
				Publish(gomock.Any(), gomock.Any()).
				Return(publishErr)

			// Mark as failed also fails
			markErr := errors.New("mark failed error")
			s.EXPECT().
				OutboxEventsMarkFailed(ctx, testEvents[0].ID, 1, publishErr).
				Return(markErr)

			err := act.OutboxPublishPendingEvents(ctx, 100)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to update event retry count"))
		})

		It("handles delete and record sent error", func(ctx SpecContext) {
			connectorID := &models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.New(),
			}

			testEvents := []models.OutboxEvent{
				{
					ID:             uuid.New(),
					EventType:      "account.saved",
					EntityID:       "acc_123",
					Payload:        json.RawMessage(`{"id":"acc_123","name":"Test Account"}`),
					CreatedAt:      time.Now().UTC(),
					Status:         models.OUTBOX_STATUS_PENDING,
					ConnectorID:    connectorID,
					RetryCount:     0,
					IdempotencyKey: "account.saved:acc_123",
				},
			}

			// Poll events
			s.EXPECT().
				OutboxEventsPollPending(ctx, 100).
				Return(testEvents, nil)

			// Publish succeeds
			mockPublisher.EXPECT().
				Publish(gomock.Any(), gomock.Any()).
				Return(nil)

			// Delete and record sent fails
			deleteErr := errors.New("delete error")
			s.EXPECT().
				OutboxEventsDeleteAndRecordSent(ctx, testEvents[0].ID, gomock.Any()).
				Return(deleteErr)

			err := act.OutboxPublishPendingEvents(ctx, 100)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to delete outbox event and record sent"))
		})
	})
})
