package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"github.com/formancehq/payments/internal/models"
)

// fetchNextBalances derives a PSPBalance from the parent PSPAccount's
// Raw payload (Qonto FromPayload pattern — no second API call).
// See MAPPINGS §4.2.
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
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to unmarshal parent account: %w", err)
	}

	balance, err := mappers.AccountBalanceToPSPBalance(currencies, parent, time.Now().UTC())
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to derive balance for %s: %w", parent.Reference, err)
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
