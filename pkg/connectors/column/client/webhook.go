package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"net/http"

	"github.com/formancehq/payments/pkg/connector/metrics"
)

type EventCategory string

const (
	EventCategoryBookTransferCompleted                EventCategory = "book.transfer.completed"
	EventCategoryBookTransferCanceled                 EventCategory = "book.transfer.canceled"
	EventCategoryBookTransferUpdated                  EventCategory = "book.transfer.updated"
	EventCategoryBookTransferHoldCreated              EventCategory = "book.transfer.hold_created"
	EventCategoryWireTransferInitiated                EventCategory = "wire.outgoing_transfer.initiated"
	EventCategoryWireTransferOutgoingCompleted        EventCategory = "wire.outgoing_transfer.completed"
	EventCategoryWireTransferIncomingCompleted        EventCategory = "wire.incoming_transfer.completed"
	EventCategoryWireTransferSubmitted                EventCategory = "wire.outgoing_transfer.submitted"
	EventCategoryWireTransferRejected                 EventCategory = "wire.outgoing_transfer.rejected"
	EventCategoryWireTransferManualReview             EventCategory = "wire.outgoing_transfer.manual_review"
	EventCategoryACHTransferInitiated                 EventCategory = "ach.outgoing_transfer.initiated"
	EventCategoryACHTransferSettled                   EventCategory = "ach.outgoing_transfer.settled"
	EventCategoryACHTransferSubmitted                 EventCategory = "ach.outgoing_transfer.submitted"
	EventCategoryACHTransferCompleted                 EventCategory = "ach.outgoing_transfer.completed"
	EventCategoryACHTransferManualReview              EventCategory = "ach.outgoing_transfer.manual_review"
	EventCategoryACHTransferReturned                  EventCategory = "ach.outgoing_transfer.returned"
	EventCategoryACHTransferCanceled                  EventCategory = "ach.outgoing_transfer.canceled"
	EventCategoryACHTransferReturnDishonored          EventCategory = "ach.outgoing_transfer.return_dishonored"
	EventCategoryACHTransferReturnContested           EventCategory = "ach.outgoing_transfer.return_contested"
	EventCategoryACHTransferNOC                       EventCategory = "ach.outgoing_transfer.noc"
	EventCategoryACHIncomingScheduled                 EventCategory = "ach.incoming_transfer.scheduled"
	EventCategoryACHIncomingSettled                   EventCategory = "ach.incoming_transfer.settled"
	EventCategoryACHIncomingNSF                       EventCategory = "ach.incoming_transfer.nsf"
	EventCategoryACHIncomingCompleted                 EventCategory = "ach.incoming_transfer.completed"
	EventCategoryACHIncomingReturned                  EventCategory = "ach.incoming_transfer.returned"
	EventCategoryACHIncomingReturnDishonored          EventCategory = "ach.incoming_transfer.return_dishonored"
	EventCategoryACHIncomingReturnContested           EventCategory = "ach.incoming_transfer.return_contested"
	EventCategoryInternationalWireCompleted           EventCategory = "swift.outgoing_transfer.completed"
	EventCategorySwiftOutgoingInitiated               EventCategory = "swift.outgoing_transfer.initiated"
	EventCategorySwiftOutgoingManualReview            EventCategory = "swift.outgoing_transfer.manual_review"
	EventCategorySwiftOutgoingSubmitted               EventCategory = "swift.outgoing_transfer.submitted"
	EventCategorySwiftOutgoingPendingReturn           EventCategory = "swift.outgoing_transfer.pending_return"
	EventCategorySwiftOutgoingReturned                EventCategory = "swift.outgoing_transfer.returned"
	EventCategorySwiftOutgoingCancellationRequested   EventCategory = "swift.outgoing_transfer.cancellation_requested"
	EventCategorySwiftOutgoingCancellationAccepted    EventCategory = "swift.outgoing_transfer.cancellation_accepted"
	EventCategorySwiftOutgoingCancellationRejected    EventCategory = "swift.outgoing_transfer.cancellation_rejected"
	EventCategorySwiftOutgoingTrackingUpdated         EventCategory = "swift.outgoing_transfer.tracking_updated"
	EventCategorySwiftIncomingInitiated               EventCategory = "swift.incoming_transfer.initiated"
	EventCategorySwiftIncomingCompleted               EventCategory = "swift.incoming_transfer.completed"
	EventCategorySwiftIncomingPendingReturn           EventCategory = "swift.incoming_transfer.pending_return"
	EventCategorySwiftIncomingReturned                EventCategory = "swift.incoming_transfer.returned"
	EventCategorySwiftIncomingCancellationRequested   EventCategory = "swift.incoming_transfer.cancellation_requested"
	EventCategorySwiftIncomingCancellationAccepted    EventCategory = "swift.incoming_transfer.cancellation_accepted"
	EventCategorySwiftIncomingCancellationRejected    EventCategory = "swift.incoming_transfer.cancellation_rejected"
	EventCategorySwiftIncomingTrackingUpdated         EventCategory = "swift.incoming_transfer.tracking_updated"
	EventCategoryRealtimeTransferCompleted            EventCategory = "realtime.outgoing_transfer.completed"
	EventCategoryRealtimeTransferInitiated            EventCategory = "realtime.outgoing_transfer.initiated"
	EventCategoryRealtimeTransferManualReview         EventCategory = "realtime.outgoing_transfer.manual_review"
	EventCategoryRealtimeTransferManualReviewApproved EventCategory = "realtime.outgoing_transfer.manual_review_approved"
	EventCategoryRealtimeTransferManualReviewRejected EventCategory = "realtime.outgoing_transfer.manual_review_rejected"
	EventCategoryRealtimeTransferRejected             EventCategory = "realtime.outgoing_transfer.rejected"
	EventCategoryRealtimeIncomingTransferCompleted    EventCategory = "realtime.incoming_transfer.completed"
)

type WebhookEvent[t any] struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
	Data      t      `json:"data"`
}

type EventSubscription struct {
	ID            string   `json:"id"`
	URL           string   `json:"url"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
	Description   string   `json:"description"`
	EnabledEvents []string `json:"enabled_events"`
	Secret        string   `json:"secret"`
	IsDisabled    bool     `json:"is_disabled"`
}

type CreateEventSubscriptionRequest struct {
	EnabledEvents []string `json:"enabled_events"`
	URL           string   `json:"url"`
}

type ListWebhookResponseWrapper[t any] struct {
	WebhookEndpoints t `json:"webhook_endpoints"`
}

func (c *client) CreateEventSubscription(ctx context.Context, es *CreateEventSubscriptionRequest) (*EventSubscription, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_event_subscription")

	body, err := json.Marshal(es)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "webhook-endpoints", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWebhookRequestFailed, err)
	}

	var res EventSubscription
	var errRes columnError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to create web hooks: %w %w", err, errRes.Error())
	}
	return &res, nil
}

func (c *client) ListEventSubscriptions(ctx context.Context) (endpoints []*EventSubscription, err error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_event_subscription")
	endpoints = make([]*EventSubscription, 0)

	for {
		req, err := c.newRequest(ctx, http.MethodGet, "webhook-endpoints", http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrWebhookRequestFailed, err)
		}

		q := req.URL.Query()
		q.Add("limit", "100")
		if len(endpoints) > 0 {
			q.Add("starting_after", endpoints[len(endpoints)-1].ID)
		}
		req.URL.RawQuery = q.Encode()

		var res ListWebhookResponseWrapper[[]*EventSubscription]
		var errRes columnError
		_, err = c.httpClient.Do(ctx, req, &res, &errRes)
		if err != nil {
			return nil, fmt.Errorf("failed to list web hooks: %w %w", err, errRes.Error())
		}

		if len(res.WebhookEndpoints) == 0 {
			break
		}
		endpoints = append(endpoints, res.WebhookEndpoints...)
	}

	return endpoints, nil
}

func (c *client) DeleteEventSubscription(ctx context.Context, eventID string) (*EventSubscription, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "delete_event_subscription")

	req, err := c.newRequest(ctx, http.MethodDelete, fmt.Sprintf("webhook-endpoints/%s", eventID), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWebhookRequestFailed, err)
	}

	var res EventSubscription
	var errRes columnError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to delete web hooks: %w %w", err, errRes.Error())
	}
	return &res, nil
}
