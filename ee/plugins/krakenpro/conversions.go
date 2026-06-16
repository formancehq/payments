package krakenpro

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/ee/plugins/krakenpro/mappers"
	"github.com/formancehq/payments/internal/models"
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

	pageSize := effectivePageSize(req.PageSize)
	start, end, ofs := state.Window.plan(nowEpoch())
	resp, err := p.client.GetLedgers(ctx, client.LedgersParams{
		Start: start, End: end, Offset: ofs, WithoutCount: true,
	})
	if err != nil {
		return models.FetchNextConversionsResponse{}, fmt.Errorf("fetch ledgers: %w", err)
	}

	// Pair conversion-typed rows by refid. A leg seen here either
	// matches a half-pair stashed in a prior cycle (emit) or is stashed
	// itself for a future cycle.
	conversions := make([]models.PSPConversion, 0, len(resp.Ledger))
	for ledgerID, entry := range resp.Ledger {
		if kind, _, _ := mappers.ClassifyLedgerType(entry.Type); kind != mappers.LedgerKindConversion {
			continue
		}
		if entry.Refid == "" {
			p.logger.WithField("ledgerID", ledgerID).Infof("conversion row has empty refid, skipping")
			continue
		}

		other, ok := state.Pending[entry.Refid]
		if !ok {
			state.Pending[entry.Refid] = pendingLeg{
				LedgerID: ledgerID, Time: entry.Time, Type: entry.Type, Subtype: entry.Subtype,
				Asset: entry.Asset, Amount: entry.Amount, Fee: entry.Fee, Balance: entry.Balance,
			}
			continue
		}
		delete(state.Pending, entry.Refid)
		if conv := p.mapConversionPair(currencies, wallets, ledgerID, entry, other); conv != nil {
			conversions = append(conversions, *conv)
		}
	}

	hasMore := state.Window.advance(len(resp.Ledger), pageSize)

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

// mapConversionPair pairs a fresh ledger row with its stashed
// half-pair into a PSPConversion. Returns nil (logged + dropped) on a
// same-sign pair or a mapping error.
func (p *Plugin) mapConversionPair(
	currencies map[string]int,
	wallets map[string]string,
	ledgerID string, entry client.LedgerEntry, other pendingLeg,
) *models.PSPConversion {
	a := mappers.ConversionLeg{LedgerID: ledgerID, Entry: entry}
	b := mappers.ConversionLeg{LedgerID: other.LedgerID, Entry: other.toLedgerEntry(entry.Refid)}
	src, dst, paired := mappers.PairConversionLegs(a, b)
	if !paired {
		p.logger.WithField("refid", entry.Refid).Infof("conversion legs have same sign, skipping")
		return nil
	}
	conv, err := mappers.ConversionPairToPSPConversion(currencies, wallets, src, dst)
	if err != nil {
		p.logger.WithField("refid", entry.Refid).Errorf("map conversion: %v", err)
		return nil
	}
	return conv
}
