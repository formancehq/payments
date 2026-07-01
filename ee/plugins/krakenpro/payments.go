package krakenpro

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/ee/plugins/krakenpro/mappers"
	"github.com/formancehq/payments/pkg/domain/models"
)

// fetchNextPayments walks /0/private/Ledgers (full type stream) and
// emits one PSPPayment per row classified as a payment. Trade /
// conversion rows are skipped here — they belong to the orders +
// conversions pipelines. Pagination is the shared frozen-end + ofs
// window (see [ledgerWindow] and MAPPINGS §3); §7 has the field map.
func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var state paymentsState
	if len(req.State) > 0 {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextPaymentsResponse{}, fmt.Errorf("unmarshal payments state: %w", err)
		}
	}

	currencies, _, err := p.ensureAssets(ctx)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	start, end, ofs := state.Window.plan(nowEpoch())
	resp, err := p.client.GetLedgers(ctx, client.LedgersParams{
		Start: start, End: end, Offset: ofs, WithoutCount: true,
	})
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("fetch ledgers: %w", err)
	}

	// Map the page; if any row's asset is missing from the cache (likely
	// listed after the last refresh), force ONE refresh and re-map before
	// the watermark advances, so the row isn't permanently skipped.
	payments, unknown := p.mapLedgerPayments(currencies, resp.Ledger)
	if len(unknown) > 0 {
		if err := p.forceRefreshAssets(ctx); err != nil {
			return models.FetchNextPaymentsResponse{}, fmt.Errorf("refresh assets for unknown payment asset: %w", err)
		}
		payments, unknown = p.mapLedgerPayments(p.snapshotAssets(), resp.Ledger)
		if len(unknown) > 0 {
			p.logger.WithField("assets", unknown).
				Errorf("payments: assets still unknown after cache refresh, skipping rows")
		}
	}

	// Compare against Kraken's fixed page size, not req.PageSize: Ledgers
	// sends no count param and always returns a server-capped page, so a
	// larger requested size would make a full page look short and skip rows.
	hasMore := state.Window.advance(len(resp.Ledger), PAGE_SIZE)

	payload, err := json.Marshal(state)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("marshal payments state: %w", err)
	}

	p.logCycle("fetch_payments", len(payments), len(resp.Ledger), state.Window, hasMore)
	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

// mapLedgerPayments maps a ledger page to PSPPayments and returns the
// assets of rows skipped because they're missing from the cache (so the
// caller can refresh + retry). Other skips (trade/conversion rows) are
// silent; mapping errors are logged and dropped.
func (p *Plugin) mapLedgerPayments(currencies map[string]int, ledger map[string]client.LedgerEntry) ([]models.PSPPayment, []string) {
	payments := make([]models.PSPPayment, 0, len(ledger))
	var unknown []string
	for ledgerID, entry := range ledger {
		res, mapErr := mappers.LedgerEntryToPSPPayment(currencies, ledgerID, entry)
		if mapErr != nil {
			p.logger.WithField("ledgerID", ledgerID).Errorf("map payment: %v", mapErr)
			continue
		}
		if res.UnknownAsset {
			unknown = append(unknown, entry.Asset)
			continue
		}
		if res.Skip || res.Payment == nil {
			continue
		}
		if res.UnknownType {
			p.logger.WithField("ledgerID", ledgerID).WithField("type", entry.Type).
				Infof("emitting PAYMENT_TYPE_OTHER for previously-unseen Kraken ledger type")
		}
		payments = append(payments, *res.Payment)
	}
	return payments, unknown
}
