package moov

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/moov/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {

	var from models.PSPAccount

	if req.FromPayload != nil {
		if err := json.Unmarshal(req.FromPayload, &from); err != nil {
			return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to unmarshal from payload: %w", err)
		}
	}

	accountID := models.ExtractNamespacedMetadata(from.Metadata, client.MoovAccountIDMetadataKey)

	wallet, err := p.client.GetWallet(ctx, accountID, from.Reference)
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to fetch wallet: %w", err)
	}

	balances := []models.PSPBalance{
		{
			Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, wallet.AvailableBalance.Currency),
			Amount:           big.NewInt(wallet.AvailableBalance.Value),
			AccountReference: wallet.WalletID,
			CreatedAt:        time.Now().UTC(),
		},
	}

	return models.FetchNextBalancesResponse{
		Balances: balances,
		HasMore:  false,
	}, nil
}
