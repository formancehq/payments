package moov

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/moov/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/moovfinancial/moov-go/pkg/moov"
)

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {

	var from moov.Account

	if req.FromPayload != nil {
		if err := json.Unmarshal(req.FromPayload, &from); err != nil {
			return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to unmarshal from payload: %w", err)
		}
	}

	moovWallets, err := p.client.GetWallets(ctx, from.AccountID)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to fetch wallets: %w", err)

	}

	accounts, err := p.fillAccounts(moovWallets, from)

	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		HasMore:  false, // wallets are not paginated
	}, nil
}

func (p *Plugin) fillAccounts(
	moovWallets []moov.Wallet,
	from moov.Account,
) ([]models.PSPAccount, error) {
	accounts := make([]models.PSPAccount, 0, len(moovWallets))

	for _, wallet := range moovWallets {
		raw, err := json.Marshal(wallet)

		if err != nil {
			return nil, fmt.Errorf("failed to marshal wallet: %w", err)
		}

		accounts = append(accounts, models.PSPAccount{
			Reference:    wallet.WalletID,
			Name:         &from.DisplayName,
			DefaultAsset: pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, wallet.AvailableBalance.Currency)),
			CreatedAt:    from.CreatedOn.UTC().Round(time.Second),
			Raw:          raw,
			Metadata: map[string]string{
				client.MoovWalletCurrencyMetadataKey: wallet.AvailableBalance.Currency,
				client.MoovWalletValueMetadataKey:    fmt.Sprintf("%d", wallet.AvailableBalance.Value),
				client.MoovValueDecimalMetadataKey:   wallet.AvailableBalance.ValueDecimal,
				client.MoovAccountIDMetadataKey:      from.AccountID,
			},
		})
	}

	return accounts, nil
}
