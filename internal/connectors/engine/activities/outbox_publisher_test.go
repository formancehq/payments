package activities_test

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	internalevents "github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/formancehq/payments/pkg/events"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("OutboxPublishPendingEvents", func() {
	var (
		act           activities.Activities
		s             *storage.MockStorage
		evts          *internalevents.Events
		mockPublisher *activities.MockPublisher
		logger        = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		s = storage.NewMockStorage(ctrl)
		mockPublisher = activities.NewMockPublisher(ctrl)
		evts = internalevents.New(mockPublisher, "http://localhost")
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
					ID: models.EventID{
						EventIdempotencyKey: "account.saved:acc_123",
						ConnectorID:         connectorID,
					},
					EventType:   events.EventTypeSavedAccounts,
					EntityID:    "acc_123",
					Payload:     json.RawMessage(`{"id":"acc_123","name":"Test Account"}`),
					CreatedAt:   time.Now().UTC(),
					Status:      models.OUTBOX_STATUS_PENDING,
					ConnectorID: connectorID,
					RetryCount:  0,
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

			// Mark as processed and record sent atomically (batch)
			s.EXPECT().
				OutboxEventsMarkProcessedAndRecordSent(ctx, gomock.Any(), gomock.Any()).
				Do(func(_ context.Context, eventIDs []models.EventID, eventsSent []models.EventSent) {
					Expect(eventIDs).To(HaveLen(1))
					Expect(eventIDs[0]).To(Equal(testEvents[0].ID))
					Expect(eventsSent).To(HaveLen(1))
					Expect(eventsSent[0].ConnectorID).To(Equal(connectorID))
					Expect(eventsSent[0].ID).To(Equal(testEvents[0].ID))
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
					ID: models.EventID{
						EventIdempotencyKey: "account.saved:acc_123",
						ConnectorID:         connectorID,
					},
					EventType:   "account.saved",
					EntityID:    "acc_123",
					Payload:     json.RawMessage(`{"id":"acc_123","name":"Test Account"}`),
					CreatedAt:   time.Now().UTC(),
					Status:      models.OUTBOX_STATUS_PENDING,
					ConnectorID: connectorID,
					RetryCount:  models.MaxOutboxRetries, // Already at max retries
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
					ID: models.EventID{
						EventIdempotencyKey: "account.saved:acc_123",
						ConnectorID:         connectorID,
					},
					EventType:   "account.saved",
					EntityID:    "acc_123",
					Payload:     json.RawMessage(`{"id":"acc_123","name":"Test Account"}`),
					CreatedAt:   time.Now().UTC(),
					Status:      models.OUTBOX_STATUS_PENDING,
					ConnectorID: connectorID,
					RetryCount:  2, // Below max retries
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

		It("publishes even unknown event type", func(ctx SpecContext) {
			connectorID := &models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.New(),
			}

			testEvents := []models.OutboxEvent{
				{
					ID: models.EventID{
						EventIdempotencyKey: "unknown.event:some_id",
						ConnectorID:         connectorID,
					},
					EventType:   "unknown.event",
					EntityID:    "some_id",
					Payload:     json.RawMessage(`{}`),
					CreatedAt:   time.Now().UTC(),
					Status:      models.OUTBOX_STATUS_PENDING,
					ConnectorID: connectorID,
					RetryCount:  0,
				},
			}

			// Poll events
			s.EXPECT().
				OutboxEventsPollPending(ctx, 100).
				Return(testEvents, nil)

			mockPublisher.EXPECT().
				Publish(gomock.Any(), gomock.Any()).
				Return(nil)

			// Mark as processed and record sent atomically (batch)
			s.EXPECT().
				OutboxEventsMarkProcessedAndRecordSent(ctx, gomock.Any(), gomock.Any()).
				Do(func(_ context.Context, eventIDs []models.EventID, eventsSent []models.EventSent) {
					Expect(eventIDs).To(HaveLen(1))
					Expect(eventIDs[0]).To(Equal(testEvents[0].ID))
					Expect(eventsSent).To(HaveLen(1))
					Expect(eventsSent[0].ConnectorID).To(Equal(connectorID))
					Expect(eventsSent[0].ID).To(Equal(testEvents[0].ID))
				}).
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
					EventType:   "account.saved",
					EntityID:    "acc_123",
					Payload:     json.RawMessage(`{"id":"acc_123","name":"Test Account"}`),
					CreatedAt:   time.Now().UTC(),
					Status:      models.OUTBOX_STATUS_PENDING,
					ConnectorID: connectorID,
					RetryCount:  0,
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
					ID: models.EventID{
						EventIdempotencyKey: "account.saved:acc_123",
						ConnectorID:         connectorID,
					},
					EventType:   "account.saved",
					EntityID:    "acc_123",
					Payload:     json.RawMessage(`{"id":"acc_123","name":"Test Account"}`),
					CreatedAt:   time.Now().UTC(),
					Status:      models.OUTBOX_STATUS_PENDING,
					ConnectorID: connectorID,
					RetryCount:  0,
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

			// Mark as processed and record sent fails
			markErr := errors.New("mark processed error")
			s.EXPECT().
				OutboxEventsMarkProcessedAndRecordSent(ctx, gomock.Any(), gomock.Any()).
				Return(markErr)

			err := act.OutboxPublishPendingEvents(ctx, 100)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to mark outbox events as processed and record sent"))
		})

		It("processes multiple events in batch", func(ctx SpecContext) {
			connectorID := &models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.New(),
			}

			testEvents := []models.OutboxEvent{
				{
					ID: models.EventID{
						EventIdempotencyKey: "account.saved:acc_123",
						ConnectorID:         connectorID,
					},
					EventType:   events.EventTypeSavedAccounts,
					EntityID:    "acc_123",
					Payload:     json.RawMessage(`{"id":"acc_123","name":"Test Account 1"}`),
					CreatedAt:   time.Now().UTC(),
					Status:      models.OUTBOX_STATUS_PENDING,
					ConnectorID: connectorID,
					RetryCount:  0,
				},
				{
					ID: models.EventID{
						EventIdempotencyKey: "account.saved:acc_456",
						ConnectorID:         connectorID,
					},
					EventType:   events.EventTypeSavedAccounts,
					EntityID:    "acc_456",
					Payload:     json.RawMessage(`{"id":"acc_456","name":"Test Account 2"}`),
					CreatedAt:   time.Now().UTC(),
					Status:      models.OUTBOX_STATUS_PENDING,
					ConnectorID: connectorID,
					RetryCount:  0,
				},
				{
					ID: models.EventID{
						EventIdempotencyKey: "account.saved:acc_789",
						ConnectorID:         connectorID,
					},
					EventType:   events.EventTypeSavedAccounts,
					EntityID:    "acc_789",
					Payload:     json.RawMessage(`{"id":"acc_789","name":"Test Account 3"}`),
					CreatedAt:   time.Now().UTC(),
					Status:      models.OUTBOX_STATUS_PENDING,
					ConnectorID: connectorID,
					RetryCount:  0,
				},
			}

			// Poll events
			s.EXPECT().
				OutboxEventsPollPending(ctx, 100).
				Return(testEvents, nil)

			// Publish all events successfully
			mockPublisher.EXPECT().
				Publish(gomock.Any(), gomock.Any()).
				Return(nil).
				Times(3)

			// Batch mark as processed and record sent
			s.EXPECT().
				OutboxEventsMarkProcessedAndRecordSent(ctx, gomock.Any(), gomock.Any()).
				Do(func(_ context.Context, eventIDs []models.EventID, eventsSent []models.EventSent) {
					Expect(eventIDs).To(HaveLen(3))
					Expect(eventsSent).To(HaveLen(3))
					Expect(eventIDs[0]).To(Equal(testEvents[0].ID))
					Expect(eventIDs[1]).To(Equal(testEvents[1].ID))
					Expect(eventIDs[2]).To(Equal(testEvents[2].ID))
					Expect(eventsSent[0].ID).To(Equal(testEvents[0].ID))
					Expect(eventsSent[1].ID).To(Equal(testEvents[1].ID))
					Expect(eventsSent[2].ID).To(Equal(testEvents[2].ID))
				}).
				Return(nil)

			err := act.OutboxPublishPendingEvents(ctx, 100)
			Expect(err).To(BeNil())
		})

		It("only batches successful events", func(ctx SpecContext) {
			connectorID := &models.ConnectorID{
				Provider:  "stripe",
				Reference: uuid.New(),
			}

			testEvents := []models.OutboxEvent{
				{
					ID: models.EventID{
						EventIdempotencyKey: "account.saved:acc_123",
						ConnectorID:         connectorID,
					},
					EventType:   events.EventTypeSavedAccounts,
					EntityID:    "acc_123",
					Payload:     json.RawMessage(`{"id":"acc_123","name":"Test Account 1"}`),
					CreatedAt:   time.Now().UTC(),
					Status:      models.OUTBOX_STATUS_PENDING,
					ConnectorID: connectorID,
					RetryCount:  0,
				},
				{
					ID: models.EventID{
						EventIdempotencyKey: "account.saved:acc_456",
						ConnectorID:         connectorID,
					},
					EventType:   events.EventTypeSavedAccounts,
					EntityID:    "acc_456",
					Payload:     json.RawMessage(`{"id":"acc_456","name":"Test Account 2"}`),
					CreatedAt:   time.Now().UTC(),
					Status:      models.OUTBOX_STATUS_PENDING,
					ConnectorID: connectorID,
					RetryCount:  0,
				},
			}

			// Poll events
			s.EXPECT().
				OutboxEventsPollPending(ctx, 100).
				Return(testEvents, nil)

			// First event publishes successfully
			mockPublisher.EXPECT().
				Publish(gomock.Any(), gomock.Any()).
				Return(nil)

			// Second event fails to publish
			publishErr := errors.New("publish error")
			mockPublisher.EXPECT().
				Publish(gomock.Any(), gomock.Any()).
				Return(publishErr)

			// Mark second event as failed
			s.EXPECT().
				OutboxEventsMarkFailed(ctx, testEvents[1].ID, 1, publishErr).
				Return(nil)

			// Only first event should be batched for marking as processed
			s.EXPECT().
				OutboxEventsMarkProcessedAndRecordSent(ctx, gomock.Any(), gomock.Any()).
				Do(func(_ context.Context, eventIDs []models.EventID, eventsSent []models.EventSent) {
					Expect(eventIDs).To(HaveLen(1))
					Expect(eventIDs[0]).To(Equal(testEvents[0].ID))
					Expect(eventsSent).To(HaveLen(1))
					Expect(eventsSent[0].ID).To(Equal(testEvents[0].ID))
				}).
				Return(nil)

			err := act.OutboxPublishPendingEvents(ctx, 100)
			Expect(err).To(BeNil())
		})
	})
})
