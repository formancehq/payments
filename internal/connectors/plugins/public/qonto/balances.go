package qonto

import (
	"context"
	"encoding/json"
	"github.com/formancehq/payments/internal/connectors/plugins/public/qonto/client"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextBalancesResponse{}, models.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	var qontoBankAccount client.OrganizationBankAccount

	if err := json.Unmarshal(from.Raw, &qontoBankAccount); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	accountBalance := models.PSPBalance{
		AccountReference: from.Reference,
		Amount:           big.NewInt(qontoBankAccount.BalanceCents),
		Asset:            *from.DefaultAsset,
		CreatedAt:        time.Now(),
	}
	accountBalances := []models.PSPBalance{accountBalance}

	return models.FetchNextBalancesResponse{
		Balances: accountBalances,
		HasMore:  false,
	}, nil
}
