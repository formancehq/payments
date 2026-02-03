package fireblocks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextBalancesResponse{}, errorsutils.NewWrappedError(
			fmt.Errorf("from payload is required"),
			models.ErrInvalidRequest,
		)
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	// Fetch fresh balance data from the API
	vaultAccount, err := p.client.GetVaultAccount(ctx, from.Reference)
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to get vault account %s: %w", from.Reference, err)
	}

	now := time.Now()
	assetDecimals := p.getAssetDecimals()
	var balances []models.PSPBalance
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

		balances = append(balances, models.PSPBalance{
			AccountReference: from.Reference,
			CreatedAt:        now,
			Amount:           amount,
			Asset:            currency.FormatAsset(assetDecimals, asset.ID),
		})
	}

	return models.FetchNextBalancesResponse{
		Balances: balances,
		HasMore:  false,
	}, nil
}
