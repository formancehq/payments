package wise

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/payments/pkg/domain/models"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextBalancesResponse{}, models.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	balanceID, err := strconv.ParseUint(from.Reference, 10, 64)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	pID, ok := from.Metadata[metadataProfileIDKey]
	if !ok {
		return models.FetchNextBalancesResponse{}, errors.New("missing profile ID in from payload when fetching balances")
	}

	profileID, err := strconv.ParseUint(pID, 10, 64)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	balance, err := p.client.GetBalance(ctx, profileID, balanceID)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	precision, ok := supportedCurrenciesWithDecimal[balance.Amount.Currency]
	if !ok {
		// Skip unsupported currencies rather than failing: a retryable error
		// here would freeze balance ingestion for the account.
		p.logger.WithField("currency", balance.Amount.Currency).Info("skipping balance with unsupported currency")
		return models.FetchNextBalancesResponse{HasMore: false}, nil
	}

	amount, err := currency.GetAmountWithPrecisionFromString(balance.Amount.Value.String(), precision)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	return models.FetchNextBalancesResponse{
		Balances: []models.PSPBalance{
			{
				AccountReference: from.Reference,
				CreatedAt:        balance.ModificationTime,
				Amount:           amount,
				Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, balance.Amount.Currency),
			},
		},
		HasMore: false,
	}, nil
}
