package client

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/engine/workflow"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/stripe/stripe-go/v79"
)

const (
	expandSource                    = "data.source"
	expandSourceCharge              = "data.source.charge"
	expandSourceDispute             = "data.source.dispute"
	expandSourcePayout              = "data.source.payout"
	expandSourceRefund              = "data.source.refund"
	expandSourceTransfer            = "data.source.transfer"
	expandSourcePaymentIntent       = "data.source.payment_intent"
	expandSourceRefundPaymentIntent = "data.source.refund.payment_intent"
)

func (c *client) GetPayments(
	ctx context.Context,
	accountID string,
	timeline Timeline,
	pageSize int64,
) (results []*stripe.BalanceTransaction, _ Timeline, hasMore bool, err error) {
	results = make([]*stripe.BalanceTransaction, 0, int(pageSize))

	timer := time.NewTimer((workflow.StartToCloseTimeoutMinutesLong - 1) * time.Minute)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			// if the app is shutting down
			return results, timeline, false, fmt.Errorf("context closed before first payment found")
		case <-timer.C:
			// after the timer expires let's save the state to prevent workflow timeout
			return results, timeline, true, nil
		default: //fallthrough
		}

		if timeline.IsCaughtUp() {
			break
		}

		var oldest interface{}
		oldest, timeline, hasMore, err = scanForOldest(timeline, pageSize, func(params stripe.ListParams) (stripe.ListContainer, error) {
			if accountID != "" {
				params.StripeAccount = &accountID
			}
			params.Context = metrics.OperationContext(ctx, "list_transactions_scan")
			transactionParams := &stripe.BalanceTransactionListParams{ListParams: params}
			expandBalanceTransactionParams(transactionParams)
			itr := c.balanceTransactionClient.List(transactionParams)
			return itr.BalanceTransactionList(), wrapSDKErr(itr.Err())
		})
		if err != nil {
			return results, timeline, false, err
		}

		if hasMore {
			continue
		}

		if oldest == nil {
			return results, timeline, false, err
		}
		results = append(results, oldest.(*stripe.BalanceTransaction))
	}

	filters := stripe.ListParams{
		Context:      metrics.OperationContext(ctx, "list_transactions"),
		Limit:        limit(pageSize, len(results)),
		EndingBefore: &timeline.LatestID,
		Single:       true, // turn off autopagination
	}

	if accountID != "" {
		filters.StripeAccount = &accountID
	}

	params := &stripe.BalanceTransactionListParams{
		ListParams: filters,
	}
	expandBalanceTransactionParams(params)

	itr := c.balanceTransactionClient.List(params)
	data := reverseTransactions(itr.BalanceTransactionList().Data)
	results = append(results, data...)
	if len(results) == 0 {
		return results, timeline, itr.BalanceTransactionList().ListMeta.HasMore, wrapSDKErr(itr.Err())
	}

	timeline.LatestID = results[len(results)-1].ID
	c.logger.WithField("account", accountID).WithField("latest_id", timeline.LatestID).Debugf("set latest id after batch with %d entries", len(results))
	return results, timeline, itr.BalanceTransactionList().ListMeta.HasMore, wrapSDKErr(itr.Err())
}

func expandBalanceTransactionParams(params *stripe.BalanceTransactionListParams) {
	params.AddExpand(expandSource)
	params.AddExpand(expandSourceCharge)
	params.AddExpand(expandSourceDispute)
	params.AddExpand(expandSourcePayout)
	params.AddExpand(expandSourceRefund)
	params.AddExpand(expandSourceTransfer)
	params.AddExpand(expandSourcePaymentIntent)
	params.AddExpand(expandSourceRefundPaymentIntent)
}

// Stripe now returns data in reverse chronological order no matter which params we provide so we need to reverse the slice
func reverseTransactions(in []*stripe.BalanceTransaction) []*stripe.BalanceTransaction {
	out := make([]*stripe.BalanceTransaction, len(in))
	copy(out, in)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
