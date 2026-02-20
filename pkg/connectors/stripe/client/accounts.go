package client

import (
	"context"

	"github.com/formancehq/payments/pkg/connector/metrics"
	"github.com/stripe/stripe-go/v80"
)

func (c *client) GetRootAccount() (result *stripe.Account, err error) {
	result, err = c.accountClient.Get()
	return result, wrapSDKErr(err)
}

func (c *client) GetAccounts(
	ctx context.Context,
	timeline Timeline,
	pageSize int64,
) (results []*stripe.Account, _ Timeline, hasMore bool, err error) {
	results = make([]*stripe.Account, 0, int(pageSize))

	if !timeline.IsCaughtUp() {
		var backlog []interface{}
		backlog, timeline, hasMore, err = fetchBacklog(timeline, pageSize, func(params stripe.ListParams) (stripe.ListContainer, error) {
			params.Context = metrics.OperationContext(ctx, "list_accounts_scan")
			itr := c.accountClient.List(&stripe.AccountListParams{ListParams: params})
			return itr.AccountList(), wrapSDKErr(itr.Err())
		})
		if err != nil {
			return results, timeline, false, err
		}
		for _, a := range backlog {
			results = append(results, a.(*stripe.Account))
		}

		return results, timeline, hasMore, err
	}

	filters := stripe.ListParams{
		Context:      metrics.OperationContext(ctx, "list_accounts"),
		Limit:        limit(pageSize, len(results)),
		EndingBefore: &timeline.LatestID,
		Single:       true, // turn off autopagination
	}

	itr := c.accountClient.List(&stripe.AccountListParams{ListParams: filters})
	data := reverseAccounts(itr.AccountList().Data)
	results = append(results, data...)
	if len(results) == 0 {
		return results, timeline, itr.AccountList().ListMeta.HasMore, wrapSDKErr(itr.Err())
	}

	timeline.LatestID = results[len(results)-1].ID
	return results, timeline, itr.AccountList().ListMeta.HasMore, wrapSDKErr(itr.Err())
}

// Stripe now returns data in reverse chronological order no matter which params we provide so we need to reverse the slice
func reverseAccounts(in []*stripe.Account) []*stripe.Account {
	out := make([]*stripe.Account, len(in))
	copy(out, in)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
