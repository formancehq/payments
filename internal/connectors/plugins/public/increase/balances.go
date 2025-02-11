package increase

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var fromPayload struct {
		AccountID string `json:"account_id"`
	}
	if req.FromPayload == nil {
		return models.FetchNextBalancesResponse{}, models.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &fromPayload); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	balance, err := p.client.GetAccountBalance(ctx, fromPayload.AccountID)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	account, err := p.client.GetAccount(ctx, fromPayload.AccountID)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	var accountBalances []models.PSPBalance
	precision, err := currency.GetPrecision(supportedCurrenciesWithDecimal, account.Currency)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	amount, err := currency.GetAmountWithPrecisionFromString(balance.AvailableBalance.String(), precision)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	asset := currency.FormatAsset(supportedCurrenciesWithDecimal, account.Currency)

	accountBalances = append(accountBalances, models.PSPBalance{
		AccountReference: balance.AccountID,
		Amount:           amount,
		Asset:            asset,
	})

	return models.FetchNextBalancesResponse{
		Balances: accountBalances,
		HasMore:  false,
	}, nil
}
