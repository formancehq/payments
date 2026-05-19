package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"github.com/formancehq/payments/internal/models"
)

// fetchNextBalances reads the parent PSPAccount from req.FromPayload
// (delivered by the engine because TASK_FETCH_BALANCES is nested
// under TASK_FETCH_ACCOUNTS in workflow.go) and derives a PSPBalance
// from the AccountBalance JSON snapshotted on PSPAccount.Raw — no
// extra Bitstamp API call. This is the Qonto pattern; see MAPPINGS.md
// §2.1 / §3.2 and the PR #679 review thread for the rationale.
func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	currencies, err := p.getCurrencies(ctx)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	if req.FromPayload == nil {
		return models.FetchNextBalancesResponse{}, models.ErrMissingFromPayloadInRequest
	}
	var parent models.PSPAccount
	if err := json.Unmarshal(req.FromPayload, &parent); err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("unmarshal parent account: %w", err)
	}

	balance, err := mappers.AccountBalanceToPSPBalance(currencies, parent, time.Now().UTC())
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("derive balance for %s: %w", parent.Reference, err)
	}
	if balance == nil {
		p.logger.Infof("skipping balance %s: unsupported currency", parent.Reference)
		return models.FetchNextBalancesResponse{HasMore: false}, nil
	}

	return models.FetchNextBalancesResponse{
		Balances: []models.PSPBalance{*balance},
		HasMore:  false,
	}, nil
}
