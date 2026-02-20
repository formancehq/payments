package fireblocks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req connector.FetchNextBalancesRequest) (connector.FetchNextBalancesResponse, error) {
	var from connector.PSPAccount
	if req.FromPayload == nil {
		return connector.FetchNextBalancesResponse{}, connector.NewWrappedError(
			fmt.Errorf("from payload is required"),
			connector.ErrInvalidRequest,
		)
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	// Fetch fresh balance data from the API
	vaultAccount, err := p.client.GetVaultAccount(ctx, from.Reference)
	if err != nil {
		return connector.FetchNextBalancesResponse{}, fmt.Errorf("failed to get vault account %s: %w", from.Reference, err)
	}

	now := time.Now()
	assetDecimals := p.getAssetDecimals()
	var balances []connector.PSPBalance
	for _, asset := range vaultAccount.Assets {
		precision, err := currency.GetPrecision(assetDecimals, asset.ID)
		if err != nil {
			p.logger.Infof("skipping balance for unknown asset %q on account %s", asset.ID, from.Reference)
			continue
		}

		amount, err := currency.GetAmountWithPrecisionFromString(asset.Available, precision)
		if err != nil {
			p.logger.Infof("skipping balance for asset %q on account %s: failed to parse amount %q", asset.ID, from.Reference, asset.Available)
			continue
		}

		balances = append(balances, connector.PSPBalance{
			AccountReference: from.Reference,
			CreatedAt:        now,
			Amount:           amount,
			Asset:            currency.FormatAsset(assetDecimals, asset.ID),
		})
	}

	return connector.FetchNextBalancesResponse{
		Balances: balances,
		HasMore:  false,
	}, nil
}
