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
// PSPConversion per refid that has both a negative and a positive
// leg. Half-pairs are buffered across cycles via the Pending map so
// a conversion split across two pages still surfaces atomically.
// See MAPPINGS §9. Pagination is the shared frozen-end + ofs window.
func (p *Plugin) fetchNextConversions(ctx context.Context, req models.FetchNextConversionsRequest) (models.FetchNextConversionsResponse, error) {
	var state conversionsState
	if len(req.State) > 0 {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextConversionsResponse{}, fmt.Errorf("unmarshal conversions state: %w", err)
		}
	}
	if state.Pending == nil {
		state.Pending = map[string]pendingLeg{}
	}

	currencies, _, err := p.ensureAssets(ctx)
	if err != nil {
		return models.FetchNextConversionsResponse{}, err
	}
	// Spot account references (symbol -> raw spot code) for attributing
	// each conversion leg to its asset's trading account, taken from the
	// asset cache — no DB lookup. The precise variant stays in metadata.
	wallets := p.snapshotAssetCodes()

	start, end, ofs := state.Window.plan(nowEpoch())
	resp, err := p.client.GetLedgers(ctx, client.LedgersParams{
		Start: start, End: end, Offset: ofs, WithoutCount: true,
	})
	if err != nil {
		return models.FetchNextConversionsResponse{}, fmt.Errorf("fetch ledgers: %w", err)
	}

	// Process the page; if a pairing references an asset missing from the
	// cache, force ONE refresh and re-process from the same starting state
	// before the watermark advances, so legs aren't lost.
	conversions, pending, unknown := p.processConversionsPage(currencies, wallets, state.Pending, resp.Ledger)
	if len(unknown) > 0 {
		if err := p.forceRefreshAssets(ctx); err != nil {
			return models.FetchNextConversionsResponse{}, fmt.Errorf("refresh assets for unknown conversion asset: %w", err)
		}
		conversions, pending, unknown = p.processConversionsPage(p.snapshotAssets(), p.snapshotAssetCodes(), state.Pending, resp.Ledger)
		if len(unknown) > 0 {
			p.logger.WithField("assets", unknown).
				Errorf("conversions: assets still unknown after cache refresh, legs kept pending")
		}
	}
	state.Pending = pending

	// Fixed Kraken page size, not req.PageSize (see fetchNextPayments).
	hasMore := state.Window.advance(len(resp.Ledger), PAGE_SIZE)

	payload, err := json.Marshal(state)
	if err != nil {
		return models.FetchNextConversionsResponse{}, fmt.Errorf("marshal conversions state: %w", err)
	}

	// `pending` rides on top of the standard cycle log so unresolved-refid build-up is visible.
	p.logCycle("fetch_conversions", len(conversions), len(resp.Ledger), state.Window, hasMore,
		"pending", len(state.Pending))
	return models.FetchNextConversionsResponse{
		Conversions: conversions,
		NewState:    payload,
		HasMore:     hasMore,
	}, nil
}

// conversionMapStatus is the outcome of pairing two conversion legs.
type conversionMapStatus int

const (
	convOK           conversionMapStatus = iota // paired + mapped, emit
	convDrop                                    // same-sign / permanent error, drop the pair
	convUnknownAsset                            // asset missing from cache, retry after refresh
)

// processConversionsPage pairs a ledger page against a COPY of pending
// and returns the emitted conversions, the resulting pending map, and the
// assets of pairs it couldn't map because of a missing-from-cache asset.
// It never mutates the input pending map, so the caller can re-run it from
// the same starting state after a cache refresh. A pending leg is removed
// only on a successful emit (or a permanent drop), never before mapping.
func (p *Plugin) processConversionsPage(
	currencies map[string]int, wallets map[string]string,
	pending map[string]pendingLeg, ledger map[string]client.LedgerEntry,
) ([]models.PSPConversion, map[string]pendingLeg, []string) {
	next := make(map[string]pendingLeg, len(pending))
	for k, v := range pending {
		next[k] = v
	}
	conversions := make([]models.PSPConversion, 0, len(ledger))
	var unknown []string
	for ledgerID, entry := range ledger {
		if kind, _ := mappers.ClassifyLedgerType(entry.Type); kind != mappers.LedgerKindConversion {
			continue
		}
		if entry.Refid == "" {
			p.logger.WithField("ledgerID", ledgerID).Infof("conversion row has empty refid, skipping")
			continue
		}
		other, ok := next[entry.Refid]
		if !ok {
			next[entry.Refid] = pendingLeg{
				LedgerID: ledgerID, Time: entry.Time, Type: entry.Type, Subtype: entry.Subtype,
				Aclass: entry.Aclass, Asset: entry.Asset, Amount: entry.Amount, Fee: entry.Fee, Balance: entry.Balance,
			}
			continue
		}
		conv, status := p.mapConversionPair(currencies, wallets, ledgerID, entry, other)
		switch status {
		case convOK:
			conversions = append(conversions, *conv)
			delete(next, entry.Refid)
		case convUnknownAsset:
			// Keep both legs available (other stays in next) so the
			// post-refresh re-run from the original pending re-pairs them.
			unknown = append(unknown, entry.Asset, other.Asset)
		case convDrop:
			delete(next, entry.Refid)
		}
	}
	return conversions, next, unknown
}

// mapConversionPair pairs a fresh ledger row with its stashed half-pair.
// It reports convDrop for a same-sign pair or a permanent mapping error,
// convUnknownAsset when a leg's asset is missing from the cache (the
// caller refreshes + retries), and convOK with the conversion otherwise.
func (p *Plugin) mapConversionPair(
	currencies map[string]int,
	wallets map[string]string,
	ledgerID string, entry client.LedgerEntry, other pendingLeg,
) (*models.PSPConversion, conversionMapStatus) {
	a := mappers.ConversionLeg{LedgerID: ledgerID, Entry: entry}
	b := mappers.ConversionLeg{LedgerID: other.LedgerID, Entry: other.toLedgerEntry(entry.Refid)}
	src, dst, paired := mappers.PairConversionLegs(a, b)
	if !paired {
		p.logger.WithField("refid", entry.Refid).Infof("conversion legs have same sign, skipping")
		return nil, convDrop
	}
	if !assetKnown(currencies, src.Entry.Asset) || !assetKnown(currencies, dst.Entry.Asset) {
		return nil, convUnknownAsset
	}
	conv, err := mappers.ConversionPairToPSPConversion(currencies, wallets, src, dst)
	if err != nil {
		p.logger.WithField("refid", entry.Refid).Errorf("map conversion: %v", err)
		return nil, convDrop
	}
	return conv, convOK
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
