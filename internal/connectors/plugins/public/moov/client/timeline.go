package client

import (
	"context"
	"time"

	"github.com/moovfinancial/moov-go/pkg/moov"
)

// Timeline allows the client to navigate the backlog and decide whether to fetch
// historical or recently added data
type Timeline struct {
	StartAtCreatedOn time.Time `json:"start_at_created_on"`
	StartAtID        string    `json:"start_at_id"`

	OldestCreatedOn time.Time `json:"oldest_created_on"`
	OldestID        string    `json:"oldest_id"`
}

func (t *Timeline) IsCaughtUp() bool {
	return !t.OldestCreatedOn.IsZero()
}

/**
* We scan backwards to the oldest transfer in the account,
* we use skip and count to go back in time since this is the default behavior of the moov api
 */
func (c *client) scanForOldest(ctx context.Context, accountID string, status moov.TransferStatus, skip int, count int, timeline Timeline) (
	[]moov.Transfer,
	Timeline,
	int,
	bool,
	error) {
	filters := []moov.ListTransferFilter{
		moov.Count(count),
		moov.WithTransactionStatus(string(status)),
		moov.Skip(skip),
	}

	transfers, err := c.service.GetMoovTransfers(ctx, accountID, filters...)
	if err != nil {
		return []moov.Transfer{}, Timeline{}, 0, false, err
	}

	// we have reached the beginning of time
	if len(transfers) == 0 {
		return []moov.Transfer{}, timeline, 0, false, nil
	}

	// if we have not set the startAtCreatedOn, we need to set it to the latest transfer
	if timeline.StartAtCreatedOn.IsZero() {
		timeline.StartAtCreatedOn = transfers[0].CreatedOn
		timeline.StartAtID = transfers[0].TransferID
	}

	hasMore := len(transfers) == count

	reverseTransfers := reverseTransactions(transfers)

	if !hasMore {
		// we've reached the beginning of time: we can return the oldest entry as the starting point for subsequent searches
		timeline.OldestCreatedOn = timeline.StartAtCreatedOn
		timeline.OldestID = timeline.StartAtID
		return reverseTransfers, timeline, 0, false, nil
	}

	skip = len(reverseTransfers) + skip
	return reverseTransfers, timeline, skip, hasMore, nil
}
