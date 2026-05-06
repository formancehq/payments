package routable

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/models"
)

// fetchNextBalances looks up the freshly-listed settings account so the
// engine sees a current balance. Routable does not expose a streaming
// balance endpoint; type_details.available_amount is the source of truth.
//
// We refetch the account by ID to surface the latest balance even if the
// engine reuses a stale FromPayload from a previous accounts cycle.
func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	if req.FromPayload == nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("missing from payload when fetching balances")
	}
	var from models.PSPAccount
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("decoding from payload: %w", err)
	}

	account, err := p.client.GetAccount(ctx, from.Reference)
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("getting account %s: %w", from.Reference, err)
	}

	balance, err := p.accountToBalance(*account)
	if err != nil {
		p.logger.Infof("skipping balance for account %s: %v", from.Reference, err)
		return models.FetchNextBalancesResponse{HasMore: false}, nil
	}

	return models.FetchNextBalancesResponse{Balances: []models.PSPBalance{balance}, HasMore: false}, nil
}

// accountToBalance converts a Routable settings account's available amount
// into a PSPBalance. Pending balances are intentionally not surfaced as a
// second entry: the Formance balance model represents one snapshot per
// (account, asset) pair, and "available" is the canonical signal.
func (p *Plugin) accountToBalance(a client.Account) (models.PSPBalance, error) {
	currencyCode := a.CurrencyCode
	if currencyCode == "" {
		// Routable historically defaulted USD here; keep the same fallback
		// behaviour rather than dropping balances silently.
		currencyCode = "USD"
	}
	precision, err := precisionFor(currencyCode)
	if err != nil {
		return models.PSPBalance{}, err
	}
	amount, err := toMinorUnits(a.TypeDetails.AvailableAmount, precision)
	if err != nil {
		return models.PSPBalance{}, fmt.Errorf("parsing available_amount: %w", err)
	}
	return models.PSPBalance{
		AccountReference: a.ID,
		Asset:            formatAsset(currencyCode),
		Amount:           amount,
		CreatedAt:        time.Now().UTC(),
	}, nil
}
