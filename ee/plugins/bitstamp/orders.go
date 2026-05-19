package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"github.com/formancehq/payments/internal/models"
)

// fetchNextOrders reconciles open orders with the previously-tracked
// set and emits one PSPOrder per order touched this cycle. See
// MAPPINGS.md §3.4 for the full lifecycle:
//
//  1. snapshot open_orders/all/
//  2. for each snapshot ID not already tracked -> seed first-sight params
//  3. for every ID in (snapshot ∪ tracked) -> GetOrderStatus to refresh
//  4. emit one PSPOrder per ID; drop terminal orders from tracked
//  5. evict any tracked entry whose FirstSeenAt + orderRetentionMax has
//     passed, with metadata com.bitstamp.spec/retention_expired
//
// All bookkeeping flows through ordersState so a worker crash mid-cycle
// resumes deterministically from the last persisted snapshot.
func (p *Plugin) fetchNextOrders(ctx context.Context, req models.FetchNextOrdersRequest) (models.FetchNextOrdersResponse, error) {
	currencies, err := p.getCurrencies(ctx)
	if err != nil {
		return models.FetchNextOrdersResponse{}, err
	}

	state, err := decodeOrdersState(req.State)
	if err != nil {
		return models.FetchNextOrdersResponse{}, err
	}

	openOrders, err := p.client.GetOpenOrders(ctx)
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("fetch open orders: %w", err)
	}

	now := time.Now().UTC()
	tracked := mergeOpenOrders(state.TrackedOrders, openOrders, now)
	openIDs := openOrderIDs(openOrders)
	ids, evicted := reconciliationIDs(tracked, openIDs, now)

	orders := make([]models.PSPOrder, 0, len(ids))
	for _, id := range ids {
		t := tracked[id]
		retentionExpired := evicted[id]

		status, err := p.client.GetOrderStatus(ctx, id)
		if err != nil {
			// A single bad order_status call must not poison the whole
			// cycle (a stale ID past 30-day retention will fail too,
			// for instance). Log and skip; the next cycle will retry
			// any still-tracked entries.
			p.logger.WithField("orderID", id).Errorf("get order status: %v", err)
			continue
		}

		order, err := mappers.OrderStatusToPSPOrder(currencies, mappers.OrderMapInput{
			Status:           status,
			Tracked:          t.toMapperInput(),
			RetentionExpired: retentionExpired,
		})
		if err != nil {
			p.logger.WithField("orderID", id).Errorf("map order: %v", err)
			continue
		}

		if !mappers.IsKnownOrderStatus(status.Status) {
			p.logger.WithField("orderID", id).WithField("status", status.Status).
				Infof("emitting order with default OPEN status for previously-unseen Bitstamp status")
		}

		orders = append(orders, *order)

		// Drop terminal orders from tracking. Forced-evicted entries
		// also drop here even when their underlying status is not
		// terminal — their next emit would be unfetchable.
		if order.Status.IsFinal() || retentionExpired {
			delete(tracked, id)
		} else {
			// Refresh the LastStatus + FirstSeenAt-preserving copy.
			entry := tracked[id]
			entry.LastStatus = status.Status
			tracked[id] = entry
		}
	}

	state.TrackedOrders = tracked
	payload, err := json.Marshal(state)
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("marshal orders state: %w", err)
	}

	// Open orders are a server-side snapshot; once we have walked
	// (snapshot ∪ tracked) there is nothing more to do this cycle.
	return models.FetchNextOrdersResponse{
		Orders:   orders,
		NewState: payload,
		HasMore:  false,
	}, nil
}

// decodeOrdersState handles nil + empty-object states without forcing
// callers to special-case the cold start.
func decodeOrdersState(raw json.RawMessage) (ordersState, error) {
	state := ordersState{TrackedOrders: map[string]trackedOrder{}}
	if len(raw) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return ordersState{}, fmt.Errorf("unmarshal orders state: %w", err)
	}
	if state.TrackedOrders == nil {
		state.TrackedOrders = map[string]trackedOrder{}
	}
	return state, nil
}

// mergeOpenOrders adds first-sight entries for new IDs in the snapshot
// and preserves existing entries' FirstSeenAt so eviction is anchored
// to the original observation, not the latest sighting.
func mergeOpenOrders(existing map[string]trackedOrder, snapshot []client.OpenOrder, now time.Time) map[string]trackedOrder {
	out := make(map[string]trackedOrder, len(existing)+len(snapshot))
	for id, t := range existing {
		out[id] = t
	}
	for _, oo := range snapshot {
		id := oo.ID
		if id == "" {
			continue
		}
		if _, seen := out[id]; seen {
			continue
		}
		out[id] = trackedOrder{
			LastStatus:   mappers.OrderStatusOpen,
			FirstSeenAt:  now,
			Price:        oo.Price,
			Amount:       oo.Amount,
			CurrencyPair: oo.CurrencyPair,
			Type:         parseOpenOrderType(oo.Type),
		}
	}
	return out
}

// openOrderIDs flattens the snapshot to a lookup set for "still open".
func openOrderIDs(snapshot []client.OpenOrder) map[string]struct{} {
	set := make(map[string]struct{}, len(snapshot))
	for _, oo := range snapshot {
		if oo.ID != "" {
			set[oo.ID] = struct{}{}
		}
	}
	return set
}

// reconciliationIDs returns:
//   - the deterministic list of IDs to fetch order_status for this
//     cycle (every tracked ID, ordered by string for test stability);
//   - the set of IDs that exceeded orderRetentionMax and must be
//     emitted with retention_expired metadata before being dropped.
func reconciliationIDs(tracked map[string]trackedOrder, openIDs map[string]struct{}, now time.Time) ([]string, map[string]bool) {
	ids := make([]string, 0, len(tracked))
	evicted := map[string]bool{}
	for id, t := range tracked {
		ids = append(ids, id)
		_, stillOpen := openIDs[id]
		if stillOpen && now.Sub(t.FirstSeenAt) >= orderRetentionMax {
			evicted[id] = true
		}
	}
	sortStrings(ids)
	return ids, evicted
}

// sortStrings is a tiny dependency-free string sort used for
// deterministic test ordering of the reconciliation ID list.
// Standard sort.Strings would do too but keeping the no-import.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// parseOpenOrderType maps Bitstamp's open_orders.type string to the
// int form expected by trackedOrder.Type / OrderTypeIntToDirection.
// Returns -1 on unknown values; the mapper subsequently fails
// validation so unknown directions are loud.
func parseOpenOrderType(s string) int {
	switch s {
	case "0":
		return 0
	case "1":
		return 1
	default:
		// Encoded as int so JSON state remains stable; -1 surfaces as
		// ORDER_DIRECTION_UNKNOWN in mappers.OrderTypeIntToDirection.
		n, err := strconv.Atoi(s)
		if err != nil {
			return -1
		}
		return n
	}
}

// toMapperInput is the bridge between the orchestrator's trackedOrder
// (state-package type) and mappers.TrackedOrderInput (mapper-package
// type). Kept here so the mapper package has no dependency back on
// the orchestrator.
func (t trackedOrder) toMapperInput() mappers.TrackedOrderInput {
	return mappers.TrackedOrderInput{
		Price:        t.Price,
		Amount:       t.Amount,
		CurrencyPair: t.CurrencyPair,
		Type:         t.Type,
		FirstSeenAt:  t.FirstSeenAt,
	}
}
