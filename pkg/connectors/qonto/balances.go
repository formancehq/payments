package qonto

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/formancehq/payments/pkg/connectors/qonto/client"
	"github.com/formancehq/payments/pkg/connector"
	"math/big"
	"time"

)

/*
*
Qonto does not have a balance API. We get the balances at the same time as we fetch the accounts, so here we just
read the data that's present in the request and format it as necessary.
*/
func (p *Plugin) fetchNextBalances(ctx context.Context, req connector.FetchNextBalancesRequest) (connector.FetchNextBalancesResponse, error) {
	var from connector.PSPAccount
	if req.FromPayload == nil {
		return connector.FetchNextBalancesResponse{}, connector.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		err := connector.NewWrappedError(
			fmt.Errorf("failed to unmarshall FromPayload"),
			err,
		)
		return connector.FetchNextBalancesResponse{}, err
	}

	var qontoBankAccount client.OrganizationBankAccount

	if err := json.Unmarshal(from.Raw, &qontoBankAccount); err != nil {
		err := connector.NewWrappedError(
			fmt.Errorf("failed to unmarshall FromPayload.raw"),
			err,
		)
		return connector.FetchNextBalancesResponse{}, err
	}

	if from.DefaultAsset == nil {
		return connector.FetchNextBalancesResponse{}, fmt.Errorf("missing default asset")
	}

	accountBalance := connector.PSPBalance{
		AccountReference: from.Reference,
		Amount:           big.NewInt(qontoBankAccount.BalanceCents),
		Asset:            *from.DefaultAsset,
		CreatedAt:        time.Now(),
	}
	accountBalances := []connector.PSPBalance{accountBalance}

	return connector.FetchNextBalancesResponse{
		Balances: accountBalances,
		HasMore:  false,
	}, nil
}
