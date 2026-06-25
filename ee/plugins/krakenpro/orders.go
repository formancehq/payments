package krakenpro

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/ee/plugins/krakenpro/mappers"
	"github.com/formancehq/payments/pkg/domain/models"
)

// openOrdersInProcessSafetyCap bounds the in-process OpenOrders cursor
// drain so a bad cursor chain can't spin forever; on hit we log and bail
// and the next cycle restarts from page 1.
const openOrdersInProcessSafetyCap = 100

// closedOrdersClosetime filters the ClosedOrders window on the close
// timestamp, so a newly-closed order with an ancient opentm still
// surfaces in the current window.
const closedOrdersClosetime = "close"

// orderPhase labels which endpoint a batch came from in log fields
// + per-row error messages. Stable strings — downstream log queries
// rely on them.
type orderPhase string

const (
	phaseOpen   orderPhase = "open"
	phaseClosed orderPhase = "closed"
)

// fetchNextOrders runs the two-source orders pipeline (see MAPPINGS §8):
//
//  1. Drain every currently-open order via Kraken's `with_cursor`
//     paging, looping in-process. Open-orders sets are bounded.
//  2. Page closed orders through the shared frozen-end + ofs window on
//     close time (see [ledgerWindow]) — drains the whole window without
//     skips before the watermark advances.
//
// Both endpoints return cumulative per-order state, so each emission is
// the order's full picture. fetch_orders is a periodic root (not nested
// under fetch_accounts): Kraken can't filter orders by account, so a
// per-account fan-out would refetch everything N times. Source/dest
// wallet refs come from the in-memory asset cache (symbol -> raw spot
// code) — no DB lookup; an unknown symbol resolves to nil refs.
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
	wallets := p.snapshotAssetCodes()

	orders := make([]models.PSPOrder, 0)

	openDrained, openMapErrors, openCursor, err := p.drainOpenOrders(ctx, currencies, pairs, wallets, state.OpenCursor, &orders)
	if err != nil {
		return models.FetchNextOrdersResponse{}, err
	}
	state.OpenCursor = openCursor // non-empty only when the cap deferred pages

	start, end, ofs := state.Closed.plan(nowEpoch())
	closedResp, err := p.client.GetClosedOrders(ctx, client.ClosedOrdersParams{
		Trades: true, WithoutCount: true,
		Start: start, End: end, Offset: ofs, Closetime: closedOrdersClosetime,
	})
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("fetch closed orders: %w", err)
	}
	closedMapErrors := p.appendMappedOrders(currencies, pairs, wallets, closedResp.Closed, phaseClosed, &orders)

	// More work if the closed window is still draining OR the open drain
	// was deferred at the safety cap — otherwise the deferred tail starves.
	// Fixed Kraken page size, not req.PageSize (see fetchNextPayments).
	hasMore := state.Closed.advance(len(closedResp.Closed), PAGE_SIZE) || state.OpenCursor != ""

	payload, err := json.Marshal(state)
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("marshal orders state: %w", err)
	}

	p.logCycle("fetch_orders", len(orders), len(closedResp.Closed), state.Closed, hasMore,
		"openDrained", openDrained,
		"openMapErrors", openMapErrors,
		"closedMapErrors", closedMapErrors,
		"openDeferred", state.OpenCursor != "")
	return models.FetchNextOrdersResponse{
		Orders:   orders,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

// drainOpenOrders walks Kraken's cursor pagination over the currently-open
// snapshot starting from `cursor` (empty = from the top), appending mapped
// orders into out. It returns the cursor to resume from: empty when the
// snapshot is fully drained, or the next-page token when it bails at
// openOrdersInProcessSafetyCap so the next cycle can continue the tail.
func (p *Plugin) drainOpenOrders(
	ctx context.Context,
	currencies map[string]int,
	pairs map[string]client.AssetPair,
	wallets map[string]string,
	cursor string,
	out *[]models.PSPOrder,
) (drained, mapErrors int, nextCursor string, err error) {
	for page := 0; ; page++ {
		if page >= openOrdersInProcessSafetyCap {
			p.logger.WithField("cap", openOrdersInProcessSafetyCap).
				Errorf("OpenOrders drain hit safety cap, resuming from saved cursor next cycle")
			return drained, mapErrors, cursor, nil
		}
		resp, err := p.client.GetOpenOrders(ctx, client.OpenOrdersParams{
			Trades:     true,
			WithCursor: true,
			Cursor:     cursor,
			Limit:      PAGE_SIZE,
		})
		if err != nil {
			return drained, mapErrors, "", fmt.Errorf("fetch open orders: %w", err)
		}

		before := len(*out)
		mapErrors += p.appendMappedOrders(currencies, pairs, wallets, resp.Open, phaseOpen, out)
		drained += len(*out) - before

		if strings.TrimSpace(resp.Cursor.Next) == "" {
			return drained, mapErrors, "", nil
		}
		cursor = resp.Cursor.Next
	}
}

// appendMappedOrders maps each (id, OrderEntry) row into a PSPOrder and
// appends to `out`, returning the count of rows skipped on a non-fatal
// map error (already logged). Wallet resolution is best-effort: an
// asset not currently held resolves to nil account refs (the order
// still emits) rather than failing the page — see MAPPINGS §8.
func (p *Plugin) appendMappedOrders(
	currencies map[string]int,
	pairs map[string]client.AssetPair,
	wallets map[string]string,
	entries map[string]client.OrderEntry,
	phase orderPhase,
	out *[]models.PSPOrder,
) int {
	var mapErrors int
	for id, oe := range entries {
		order, err := mappers.OrderEntryToPSPOrder(currencies, pairs, wallets,
			mappers.OrderEntryWithID{OrderID: id, Order: oe})
		if err != nil {
			mapErrors++
			p.logger.WithField("orderID", id).WithField("phase", string(phase)).
				Errorf("map order: %v", err)
			continue
		}
		if order != nil {
			*out = append(*out, *order)
		}
	}
	return mapErrors
}
