package qonto

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/formancehq/payments/internal/connectors/plugins/public/qonto/client"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/models"
)

/*
*
Qonto does not have a balance API. We get the balances at the same time as we fetch the accounts, so here we just
read the data that's present in the request and format it as necessary.
*/
func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextBalancesResponse{}, models.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		err := errorsutils.NewWrappedError(
			fmt.Errorf("failed to unmarshall FromPayload"),
			err,
		)
		return models.FetchNextBalancesResponse{}, err
	}

	var qontoBankAccount client.OrganizationBankAccount

	if err := json.Unmarshal(from.Raw, &qontoBankAccount); err != nil {
		err := errorsutils.NewWrappedError(
			fmt.Errorf("failed to unmarshall FromPayload.raw"),
			err,
		)
		return models.FetchNextBalancesResponse{}, err
	}

	if from.DefaultAsset == nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("missing default asset")
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
