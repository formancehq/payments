package krakenpro

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/ee/plugins/krakenpro/mappers"
	"github.com/formancehq/payments/pkg/domain/models"
)

// fetchNextOrders pages closed orders through the shared frozen-end + ofs
// window (see [ledgerWindow]) on close time, so a newly-closed order with
// an ancient open time still surfaces in the current window. Each row
// carries cumulative per-order state, so every emission is the order's
// full picture (no aggregation across fills/pages).
//
// Only ClosedOrders is fetched: OpenOrders has no page-size limit, so a
// drain could return an unbounded set and exceed Temporal's max payload.
// ClosedOrders is page-bounded and still returns per-fill txids with
// trades:true, so fill traceability is preserved (see MAPPINGS §8).
//
// fetch_orders is a periodic root (not nested under fetch_accounts):
// Kraken can't filter orders by account, so a per-account fan-out would
// refetch everything N times. Source/dest account refs come from the
// pair's base/quote raw codes (the per-variant account references), so
// no asset cache or DB lookup is needed.
func (p *Plugin) fetchNextOrders(ctx context.Context, req models.FetchNextOrdersRequest) (models.FetchNextOrdersResponse, error) {
	var state ordersState
	if len(req.State) > 0 {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextOrdersResponse{}, fmt.Errorf("unmarshal orders state: %w", err)
		}
	}

	currencies, pairs, err := p.ensureAssets(ctx)
	if err != nil {
		return models.FetchNextOrdersResponse{}, err
	}

	start, end, ofs := state.Closed.plan(nowEpoch())
	resp, err := p.client.GetClosedOrders(ctx, client.ClosedOrdersParams{
		Trades: true, WithoutCount: true,
		Start: start, End: end, Offset: ofs, Closetime: client.ClosetimeClose,
	})
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("fetch closed orders: %w", err)
	}

	orders := make([]models.PSPOrder, 0, len(resp.Closed))
	mapErrors := p.appendMappedOrders(currencies, pairs, resp.Closed, &orders)

	// Compare against Kraken's fixed page size, not req.PageSize (see
	// fetchNextPayments).
	hasMore := state.Closed.advance(len(resp.Closed), PAGE_SIZE)

	payload, err := json.Marshal(state)
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("marshal orders state: %w", err)
	}

	p.logCycle("fetch_orders", len(orders), len(resp.Closed), state.Closed, hasMore,
		"mapErrors", mapErrors)
	return models.FetchNextOrdersResponse{
		Orders:   orders,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

// appendMappedOrders maps each (id, OrderEntry) row into a PSPOrder and
// appends to `out`, returning the count of rows skipped on a non-fatal
// map error (already logged). Account resolution is best-effort: a pair
// whose spot account isn't currently held resolves to nil refs (the
// order still emits) rather than failing the page — see MAPPINGS §8.
func (p *Plugin) appendMappedOrders(
	currencies map[string]int,
	pairs map[string]client.AssetPair,
	entries map[string]client.OrderEntry,
	out *[]models.PSPOrder,
) int {
	var mapErrors int
	for id, oe := range entries {
		order, err := mappers.OrderEntryToPSPOrder(currencies, pairs,
			mappers.OrderEntryWithID{OrderID: id, Order: oe})
		if err != nil {
			mapErrors++
			p.logger.WithField("orderID", id).Errorf("map order: %v", err)
			continue
		}
		if order != nil {
			*out = append(*out, *order)
		}
	}
	return mapErrors
}
