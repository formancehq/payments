package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"github.com/formancehq/payments/internal/models"
)

// fetchNextOrders reconciles open_orders with tracked state, polls
// order_status per tracked id, and emits one PSPOrder per cycle.
// See MAPPINGS §4.4 for the full lifecycle.
func (p *Plugin) fetchNextOrders(ctx context.Context, req models.FetchNextOrdersRequest) (models.FetchNextOrdersResponse, error) {
	state := ordersState{TrackedOrders: map[string]trackedOrder{}}
	if len(req.State) > 0 {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to unmarshal orders state: %w", err)
		}
		if state.TrackedOrders == nil {
			state.TrackedOrders = map[string]trackedOrder{}
		}
	}

	openOrders, err := p.client.GetOpenOrders(ctx)
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to fetch open orders: %w", err)
	}

	now := time.Now().UTC()
	tracked := mergeOpenOrders(state.TrackedOrders, openOrders, now)
	openIDs := make(map[string]struct{}, len(openOrders))
	for _, oo := range openOrders {
		if oo.ID != "" {
			openIDs[oo.ID] = struct{}{}
		}
	}
	ids, evicted := reconciliationIDs(tracked, openIDs, now)

	if len(ids) == 0 {
		state.TrackedOrders = tracked
		payload, err := json.Marshal(state)
		if err != nil {
			return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to marshal orders state: %w", err)
		}
		return models.FetchNextOrdersResponse{NewState: payload, HasMore: false}, nil
	}

	currencies, err := p.getCurrencies(ctx)
	if err != nil {
		return models.FetchNextOrdersResponse{}, err
	}

	orders := make([]models.PSPOrder, 0, len(ids))
	for _, id := range ids {
		t := tracked[id]
		retentionExpired := evicted[id]

		status, err := p.client.GetOrderStatus(ctx, id)
		if err != nil {
			// A stale ID past 30-day retention will reliably fail this
			// call — evict on the spot to avoid permanent retry storms
			// and unbounded state growth (logs at Info because the
			// failure is expected for retention-expired IDs).
			if retentionExpired {
				p.logger.WithField("orderID", id).
					Infof("evicting retention-expired order after order_status error: %v", err)
				delete(tracked, id)
				continue
			}
			p.logger.WithField("orderID", id).Errorf("failed to get order status: %v", err)
			continue
		}

		order, err := mappers.OrderStatusToPSPOrder(currencies, mappers.OrderMapInput{
			Status:           status,
			Tracked:          t.toMapperInput(),
			RetentionExpired: retentionExpired,
		})
		if err != nil {
			p.logger.WithField("orderID", id).Errorf("failed to map order: %v", err)
			continue
		}

		if !mappers.IsKnownOrderStatus(status.Status) {
			p.logger.WithField("orderID", id).WithField("status", status.Status).
				Infof("emitting order with default OPEN status for previously-unseen Bitstamp status")
		}

		orders = append(orders, *order)

		if order.Status.IsFinal() || retentionExpired {
			delete(tracked, id)
		} else {
			entry := tracked[id]
			entry.LastStatus = status.Status
			tracked[id] = entry
		}
	}

	state.TrackedOrders = tracked
	payload, err := json.Marshal(state)
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to marshal orders state: %w", err)
	}

	return models.FetchNextOrdersResponse{
		Orders:   orders,
		NewState: payload,
		HasMore:  false,
	}, nil
}

// mergeOpenOrders adds first-sight entries for new snapshot IDs.
// Existing entries preserve FirstSeenAt so eviction is anchored to
// the original observation, not the latest sighting.
func mergeOpenOrders(existing map[string]trackedOrder, snapshot []client.OpenOrder, now time.Time) map[string]trackedOrder {
	out := make(map[string]trackedOrder, len(existing)+len(snapshot))
	for id, t := range existing {
		out[id] = t
	}
	for _, oo := range snapshot {
		if oo.ID == "" {
			continue
		}
		if _, seen := out[oo.ID]; seen {
			continue
		}
		out[oo.ID] = trackedOrder{
			LastStatus:  mappers.OrderStatusOpen,
			FirstSeenAt: now,
			LimitPrice:  oo.Price,
		}
	}
	return out
}

// reconciliationIDs returns the sorted list of tracked IDs + the set
// past orderRetentionMax (still in the open snapshot but evictable).
func reconciliationIDs(tracked map[string]trackedOrder, openIDs map[string]struct{}, now time.Time) ([]string, map[string]bool) {
	ids := make([]string, 0, len(tracked))
	evicted := map[string]bool{}
	for id, t := range tracked {
		ids = append(ids, id)
		if _, stillOpen := openIDs[id]; stillOpen && now.Sub(t.FirstSeenAt) >= orderRetentionMax {
			evicted[id] = true
		}
	}
	sort.Strings(ids)
	return ids, evicted
}

// toMapperInput bridges the state-package trackedOrder to the
// mapper-package input shape. The slim shape — LimitPrice +
// FirstSeenAt only — reflects the live-shape order_status response.
func (t trackedOrder) toMapperInput() mappers.TrackedOrderInput {
	return mappers.TrackedOrderInput{
		Price:       t.LimitPrice,
		FirstSeenAt: t.FirstSeenAt,
	}
}
