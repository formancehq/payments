package krakenpro

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/ee/plugins/krakenpro/mappers"
	"github.com/formancehq/payments/pkg/domain/models"
)

// fetchNextConversions scans /0/private/Ledgers and emits one
// PSPConversion per refid that has both a negative and a positive leg.
// Half-pairs are buffered across cycles via the Pending map so a
// conversion split across two pages still surfaces atomically.
// See MAPPINGS §9. Pagination is the shared frozen-end + ofs window.
//
// Concerns are separated: a pre-pass guarantees every row carries a
// known asset (refreshing the cache once if needed and dropping rows
// still unknown) so the pairing step can assume valid assets, and so
// un-convertible legs never enter the persisted Pending map. Pending is
// pruned once the watermark passes a leg's time, bounding its growth.
func (p *Plugin) fetchNextConversions(ctx context.Context, req models.FetchNextConversionsRequest) (models.FetchNextConversionsResponse, error) {
	var state conversionsState
	if len(req.State) > 0 {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextConversionsResponse{}, fmt.Errorf("unmarshal conversions state: %w", err)
		}
	}
	if state.Pending == nil {
		state.Pending = map[string]client.LedgerEntry{}
	}

	currencies, _, err := p.ensureAssets(ctx)
	if err != nil {
		return models.FetchNextConversionsResponse{}, err
	}

	start, end, ofs := state.Window.plan(nowEpoch())
	resp, err := p.client.GetLedgers(ctx, client.LedgersParams{
		Start: start, End: end, Offset: ofs, WithoutCount: true,
	})
	if err != nil {
		return models.FetchNextConversionsResponse{}, fmt.Errorf("fetch ledgers: %w", err)
	}

	// Pre-pass: keep only known-asset conversion rows. A single forced
	// refresh covers assets listed after the last cache load; rows still
	// unknown are dropped (logged) so they never reach pairing or Pending.
	rows, err := p.knownConversionRows(ctx, currencies, resp.Ledger)
	if err != nil {
		return models.FetchNextConversionsResponse{}, err
	}

	conversions := p.pairConversions(p.snapshotAssets(), state.Pending, rows)

	hasMore := state.Window.advance(len(resp.Ledger), PAGE_SIZE)
	// Prune half-pairs whose window has fully drained: both legs share a
	// refid+time, so once the watermark passes a leg's time its partner
	// can no longer arrive. Bounds Pending deterministically.
	prunePending(state.Pending, state.Window.Watermark)

	payload, err := json.Marshal(state)
	if err != nil {
		return models.FetchNextConversionsResponse{}, fmt.Errorf("marshal conversions state: %w", err)
	}

	p.logCycle("fetch_conversions", len(conversions), len(resp.Ledger), state.Window, hasMore,
		"pending", len(state.Pending))
	return models.FetchNextConversionsResponse{
		Conversions: conversions,
		NewState:    payload,
		HasMore:     hasMore,
	}, nil
}

// knownConversionRows returns the page's conversion rows (ID + refid set)
// whose asset is in the cache. If any asset is missing it forces one
// cache refresh and re-checks; rows still unknown are dropped + logged so
// they never enter pairing or the persisted Pending map.
func (p *Plugin) knownConversionRows(ctx context.Context, currencies map[string]int, ledger map[string]client.LedgerEntry) ([]client.LedgerEntry, error) {
	var rows []client.LedgerEntry
	for ledgerID, entry := range ledger {
		if kind, _ := mappers.ClassifyLedgerType(entry.Type); kind != mappers.LedgerKindConversion {
			continue
		}
		if entry.Refid == "" {
			p.logger.WithField("ledgerID", ledgerID).Infof("conversion row has empty refid, skipping")
			continue
		}
		entry.ID = ledgerID
		rows = append(rows, entry)
	}

	if !allAssetsKnown(currencies, rows) {
		if err := p.forceRefreshAssets(ctx); err != nil {
			return nil, fmt.Errorf("refresh assets for unknown conversion asset: %w", err)
		}
		currencies = p.snapshotAssets()
	}

	known := rows[:0]
	for _, entry := range rows {
		if assetKnown(currencies, entry.Asset) {
			known = append(known, entry)
			continue
		}
		p.logger.WithField("ledgerID", entry.ID).WithField("asset", entry.Asset).
			Errorf("conversion asset still unknown after cache refresh, dropping row")
	}
	return known, nil
}

// pairConversions pairs known-asset rows against the pending buffer,
// mutating pending in place: a matched refid emits a PSPConversion and
// is cleared; an unmatched row is buffered for a later page/cycle.
func (p *Plugin) pairConversions(currencies map[string]int, pending map[string]client.LedgerEntry, rows []client.LedgerEntry) []models.PSPConversion {
	conversions := make([]models.PSPConversion, 0, len(rows))
	for _, entry := range rows {
		other, ok := pending[entry.Refid]
		if !ok {
			pending[entry.Refid] = entry
			continue
		}
		delete(pending, entry.Refid)
		if conv := p.mapConversionPair(currencies, entry, other); conv != nil {
			conversions = append(conversions, *conv)
		}
	}
	return conversions
}

// mapConversionPair builds a PSPConversion from two legs sharing a refid,
// or returns nil (logged) for a same-sign pair or a mapping error.
func (p *Plugin) mapConversionPair(currencies map[string]int, a, b client.LedgerEntry) *models.PSPConversion {
	src, dst, paired := mappers.PairConversionLegs(
		mappers.ConversionLeg{LedgerID: a.ID, Entry: a},
		mappers.ConversionLeg{LedgerID: b.ID, Entry: b},
	)
	if !paired {
		p.logger.WithField("refid", a.Refid).Infof("conversion legs have same sign, skipping")
		return nil
	}
	conv, err := mappers.ConversionPairToPSPConversion(currencies, src, dst)
	if err != nil {
		p.logger.WithField("refid", a.Refid).Errorf("map conversion: %v", err)
		return nil
	}
	return conv
}

// prunePending drops buffered legs whose time is at or below the committed
// watermark — their window has fully drained, so the partner leg can no
// longer arrive.
func prunePending(pending map[string]client.LedgerEntry, watermark float64) {
	for refid, leg := range pending {
		if leg.Time <= watermark {
			delete(pending, refid)
		}
	}
}

// allAssetsKnown reports whether every row's asset is in the cache.
func allAssetsKnown(currencies map[string]int, rows []client.LedgerEntry) bool {
	for _, entry := range rows {
		if !assetKnown(currencies, entry.Asset) {
			return false
		}
	}
	return true
}

// assetKnown reports whether a raw Kraken asset normalises to a symbol
// present in the cache.
func assetKnown(currencies map[string]int, asset string) bool {
	sym := mappers.NormalizeAsset(asset)
	if sym == "" {
		return false
	}
	_, ok := currencies[sym]
	return ok
}
