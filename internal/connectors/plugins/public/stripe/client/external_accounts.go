package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/stripe/stripe-go/v79"
)

func (c *client) GetExternalAccounts(
	ctx context.Context,
	accountID string,
	timeline Timeline,
	pageSize int64,
) (results []*stripe.BankAccount, _ Timeline, hasMore bool, err error) {
	results = make([]*stripe.BankAccount, 0, int(pageSize))

	// return 0 results because this endpoint cannot be used for root account
	if accountID == "" {
		return results, timeline, false, nil
	}

	if !timeline.IsCaughtUp() {
		var backlog []interface{}
		backlog, timeline, hasMore, err = fetchBacklog(timeline, pageSize, func(params stripe.ListParams) (stripe.ListContainer, error) {
			params.Context = metrics.OperationContext(ctx, "list_bank_accounts_scan")
			itr := c.bankAccountClient.List(&stripe.BankAccountListParams{
				Account:    &accountID,
				ListParams: params,
			})
			return itr.BankAccountList(), wrapSDKErr(itr.Err())
		})
		if err != nil {
			return results, timeline, false, err
		}
		for _, a := range backlog {
			results = append(results, a.(*stripe.BankAccount))
		}

		return results, timeline, hasMore, err
	}

	itr := c.bankAccountClient.List(&stripe.BankAccountListParams{
		Account: &accountID,
		ListParams: stripe.ListParams{
			Context:      metrics.OperationContext(ctx, "list_bank_accounts"),
			Limit:        &pageSize,
			EndingBefore: &timeline.LatestID,
		},
	})
	if err := itr.Err(); err != nil {
		return nil, timeline, false, wrapSDKErr(err)
	}
	data := reverseBankAccounts(itr.BankAccountList().Data)
	results = append(results, data...)
	if len(results) == 0 {
		return results, timeline, itr.BankAccountList().ListMeta.HasMore, nil
	}
	timeline.LatestID = results[len(results)-1].ID
	return results, timeline, itr.BankAccountList().ListMeta.HasMore, nil
}

// Stripe now returns data in reverse chronological order no matter which params we provide so we need to reverse the slice
func reverseBankAccounts(in []*stripe.BankAccount) []*stripe.BankAccount {
	out := make([]*stripe.BankAccount, len(in))
	copy(out, in)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
