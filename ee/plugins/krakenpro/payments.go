package krakenpro

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/ee/plugins/krakenpro/mappers"
	"github.com/formancehq/payments/internal/models"
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
	// Spot account references for attributing a payment to its asset's
	// trading account (the raw variant stays in kraken_asset metadata).
	wallets, err := p.resolveWallets(ctx)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	pageSize := effectivePageSize(req.PageSize)
	start, end, ofs := state.Window.plan(nowEpoch())
	resp, err := p.client.GetLedgers(ctx, client.LedgersParams{
		Start: start, End: end, Offset: ofs, WithoutCount: true,
	})
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("fetch ledgers: %w", err)
	}

	payments := make([]models.PSPPayment, 0, len(resp.Ledger))
	for ledgerID, entry := range resp.Ledger {
		res, mapErr := mappers.LedgerEntryToPSPPayment(currencies, wallets, ledgerID, entry)
		if mapErr != nil {
			p.logger.WithField("ledgerID", ledgerID).Errorf("map payment: %v", mapErr)
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

	hasMore := state.Window.advance(len(resp.Ledger), pageSize)

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

// effectivePageSize defaults to PAGE_SIZE when the engine passes a
// non-positive value. HasMore lives off this comparison.
func effectivePageSize(requested int) int {
	if requested <= 0 {
		return PAGE_SIZE
	}
	return requested
}
