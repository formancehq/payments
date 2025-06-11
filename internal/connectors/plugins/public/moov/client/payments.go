package client

import (
	"context"

	"github.com/moovfinancial/moov-go/pkg/moov"
)

func (c *client) GetPayments(ctx context.Context, accountID string, status moov.TransferStatus, skip int, count int, timeline Timeline) ([]moov.Transfer, Timeline, bool, int, error) {

	// First phase: scroll through the transfers until we find the oldest transfer (beginning of time)
	// During this phase, we don't return any transfers to the caller

	if !timeline.IsCaughtUp() {
		transfers, timeline, skip, hasMore, err := c.scanForOldest(ctx, accountID, status, skip, count, timeline)
		if err != nil {
			return transfers, timeline, false, skip, err
		}

		// If we're caught up, we need to set the skip to 0
		if timeline.IsCaughtUp() {
			skip = 0
		}

		return transfers, timeline, hasMore, skip, nil
	}

	lastTransfer := moov.Transfer{}

	// Second phase: Now that we've found the beginning, we can start fetching transfers in chronological order
	// We use startDateTime to fetch newer transfers from where we left off
	// we don't use `skip` while making request to Moov if we want to fetch the latest payments
	filters := []moov.ListTransferFilter{
		moov.Count(count),
		moov.Skip(skip),
		moov.WithTransactionStatus(string(status)),
		moov.WithTransferStartDate(timeline.OldestCreatedOn), // there is always a start date
	}

	results, err := c.service.GetMoovTransfers(ctx, accountID, filters...)
	if err != nil {
		return nil, timeline, false, skip, err
	}

	if len(results) == 0 {
		return nil, timeline, false, skip, nil
	}

	// check if this is the first time we are fetching transfers
	if skip == 0 {
		timeline.StartAtCreatedOn = results[0].CreatedOn
		timeline.StartAtID = results[0].TransferID
	}

	lastTransfer = results[len(results)-1]

	// check if we have reached the oldest transfer
	if lastTransfer.TransferID == timeline.OldestID {
		timeline.OldestCreatedOn = timeline.StartAtCreatedOn
		timeline.OldestID = timeline.StartAtID
		skip = 0
	}

	// Moov returns data in reverse chronological order so we need to reverse the slice
	// to get oldest-to-newest (chronological) order
	reverseTransfers := reverseTransactions(results)

	hasMore := !(len(reverseTransfers) == 1 && lastTransfer.TransferID == timeline.OldestID && timeline.OldestID != "")

	return reverseTransfers, timeline, hasMore, skip, nil
}

func reverseTransactions(in []moov.Transfer) []moov.Transfer {
	out := make([]moov.Transfer, len(in))
	copy(out, in)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
