package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"github.com/formancehq/payments/internal/models"
)

// fetchNextConversions scans the same /api/v2/user_transactions/ stream
// as payments — but on its own since_id watermark — and emits one
// PSPConversion per type-36 row with exactly two non-zero known
// currencies. Detection lives in mappers.UserTransactionToPSPConversion
// (MAPPINGS.md §3.5); the orchestrator is responsible only for
// pagination, state, and Warn-log triage of skipped rows.
//
// No dedicated client method exists for conversions — Bitstamp has no
// /api/v2/conversions/ endpoint, so adding one would create dead
// code (per fguery's #660 feedback).
func (p *Plugin) fetchNextConversions(ctx context.Context, req models.FetchNextConversionsRequest) (models.FetchNextConversionsResponse, error) {
	currencies, err := p.getCurrencies(ctx)
	if err != nil {
		return models.FetchNextConversionsResponse{}, err
	}

	var state conversionsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextConversionsResponse{}, fmt.Errorf("unmarshal conversions state: %w", err)
		}
	}

	limit := effectivePageSize(req.PageSize)
	transactions, err := p.client.GetUserTransactions(ctx, sinceIDFor(state.LastTransactionID), limit)
	if err != nil {
		return models.FetchNextConversionsResponse{}, fmt.Errorf("fetch conversions: %w", err)
	}

	conversions := make([]models.PSPConversion, 0, len(transactions))
	lastSeen := state.LastTransactionID
	for _, tx := range transactions {
		if tx.ID > lastSeen {
			lastSeen = tx.ID
		}
		res, err := mappers.UserTransactionToPSPConversion(currencies, tx)
		if err != nil {
			return models.FetchNextConversionsResponse{}, fmt.Errorf("map conversion %d: %w", tx.ID, err)
		}
		if res.DerivativesRow {
			p.logger.WithField("txID", tx.ID).Errorf("skipping derivatives-marked row on spot-only connector")
			continue
		}
		if res.Skip || res.Conversion == nil {
			continue
		}
		conversions = append(conversions, *res.Conversion)
	}

	payload, err := json.Marshal(conversionsState{LastTransactionID: lastSeen})
	if err != nil {
		return models.FetchNextConversionsResponse{}, fmt.Errorf("marshal conversions state: %w", err)
	}

	return models.FetchNextConversionsResponse{
		Conversions: conversions,
		NewState:    payload,
		HasMore:     len(transactions) == limit,
	}, nil
}
