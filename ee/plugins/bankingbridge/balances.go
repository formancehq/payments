package bankingbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/payments/ee/plugins/bankingbridge/client"
	"github.com/formancehq/payments/pkg/domain/models"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var oldState workflowState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextBalancesResponse{}, err
		}
	}

	newState := workflowState{
		Cursor:             oldState.Cursor,
		LastSeenImportedAt: oldState.LastSeenImportedAt,
	}

	balances := make([]models.PSPBalance, 0, req.PageSize)
	pagedBalances, hasMore, cursor, err := p.client.GetAccountBalances(ctx, newState.Cursor, newState.LastSeenImportedAt, req.PageSize)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	for _, balance := range pagedBalances {
		// Advance past every balance, kept or not, so a discarded one is not
		// fetched again.
		newState.LastSeenImportedAt = balance.ImportedAt.UTC().Format(ImportedAtLayout)

		pspBalance, ok, err := ToPSPBalance(balance)
		if err != nil {
			return models.FetchNextBalancesResponse{}, err
		}
		if !ok {
			continue
		}
		balances = append(balances, pspBalance)
	}

	newState.Cursor = cursor
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	return models.FetchNextBalancesResponse{
		Balances: balances,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

const (
	// balanceTypeClosingBooked is the ISO 20022 closing booked balance (CLBD).
	balanceTypeClosingBooked = "CLBD"
	// balanceTypeInterimBooked is the ISO 20022 interim booked balance (ITBD),
	// which is what open banking channels report.
	balanceTypeInterimBooked = "ITBD"
)

// ToPSPBalance converts a Banking Bridge balance, reporting false when it
// should be discarded.
//
// Banking Bridge releases that serve the legacy payments-shaped balance send
// reportedAt, which is used as is. Later releases send every balance type
// with its date instead and no reportedAt. Of those, only the booked ones are
// kept:
//   - CLBD is timestamped at midnight UTC on its balance date.
//   - ITBD is timestamped at its import, as the legacy shape dated open
//     banking balances. Not its balance date: a statement can report an ITBD
//     and a CLBD for the same date, and payments keys a balance on its
//     account, asset and timestamp, so the two would collide.
func ToPSPBalance(in client.Balance) (models.PSPBalance, bool, error) {
	createdAt := in.ReportedAt
	if createdAt.IsZero() {
		switch in.BalanceType {
		case balanceTypeClosingBooked:
			balanceDate, err := time.ParseInLocation(time.DateOnly, in.BalanceDate, time.UTC)
			if err != nil {
				return models.PSPBalance{}, false, fmt.Errorf("invalid balance date %q for account %s: %w", in.BalanceDate, in.AccountReference, err)
			}
			createdAt = balanceDate
		case balanceTypeInterimBooked:
			createdAt = in.ImportedAt.UTC()
		default:
			return models.PSPBalance{}, false, nil
		}
	}

	amount := big.NewInt(in.AmountInMinors)
	return models.PSPBalance{
		AccountReference: in.AccountReference,
		Amount:           amount,
		Asset:            in.Asset,
		CreatedAt:        createdAt,
	}, true, nil
}
