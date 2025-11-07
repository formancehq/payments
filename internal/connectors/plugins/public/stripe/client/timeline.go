package client

import (
	"fmt"

	"github.com/stripe/stripe-go/v79"
)

// Timeline allows the client to navigate the backlog and decide whether to fetch
// historical or recently added data
type Timeline struct {
	LatestID             string `json:"latest_id"`
	BacklogCursor        string `json:"backlog_cursor"`
	BacklogStartingPoint string `json:"backlog_starting_point,omitempty"` // used by users of fetchBacklog
}

func (t Timeline) IsCaughtUp() bool {
	return t.LatestID != ""
}

// used to find oldest transaction since we can only import them in chronological order
func scanForOldest(
	timeline Timeline,
	pageSize int64,
	listFn func(stripe.ListParams) (stripe.ListContainer, error),
) (interface{}, Timeline, bool, error) {
	filters := stripe.ListParams{
		Limit:  limit(pageSize, 0),
		Single: true, // turn off autopagination
	}
	if timeline.BacklogCursor != "" {
		filters.StartingAfter = &timeline.BacklogCursor
	}

	var oldest interface{}
	var oldestID string

	list, err := listFn(filters)
	if err != nil {
		return oldest, timeline, false, err
	}
	hasMore := list.GetListMeta().HasMore

	switch v := list.(type) {
	case *stripe.BalanceTransactionList:
		if len(v.Data) == 0 {
			return oldest, timeline, hasMore, nil
		}
		trx := v.Data[len(v.Data)-1]
		oldest = trx
		oldestID = trx.ID
	default:
		return nil, timeline, hasMore, fmt.Errorf("failed to scan for oldest for type %T", list)
	}

	// we haven't found the oldest yet
	if hasMore {
		timeline.BacklogCursor = oldestID
		return nil, timeline, hasMore, nil
	}
	timeline.LatestID = oldestID
	return oldest, timeline, hasMore, nil
}

// fetchBacklog receives records we don't need to fetch in chronological order
func fetchBacklog(
	timeline Timeline,
	pageSize int64,
	listFn func(stripe.ListParams) (stripe.ListContainer, error),
) ([]interface{}, Timeline, bool, error) {
	filters := stripe.ListParams{
		Limit:  limit(pageSize, 0),
		Single: true, // turn off autopagination
	}
	if timeline.BacklogCursor != "" {
		filters.StartingAfter = &timeline.BacklogCursor
	}

	results := make([]interface{}, 0, pageSize)
	var oldestID, newestID string

	list, err := listFn(filters)
	if err != nil {
		return results, timeline, false, err
	}
	hasMore := list.GetListMeta().HasMore

	switch v := list.(type) {
	case *stripe.AccountList:
		if len(v.Data) == 0 {
			return results, timeline, hasMore, nil
		}
		account := v.Data[len(v.Data)-1]
		oldestID = account.ID
		newestID = v.Data[0].ID
		for _, acc := range v.Data {
			results = append(results, acc)
		}

	case *stripe.BankAccountList:
		if len(v.Data) == 0 {
			return results, timeline, hasMore, nil
		}
		account := v.Data[len(v.Data)-1]
		oldestID = account.ID
		newestID = v.Data[0].ID
		for _, acc := range v.Data {
			results = append(results, acc)
		}
	default:
		return results, timeline, hasMore, fmt.Errorf("failed to fetch backlog for type %T", list)
	}

	// on the very first run we keep track of where we started
	if timeline.BacklogStartingPoint == "" {
		timeline.BacklogStartingPoint = newestID
	}

	// we haven't found the oldest yet
	if hasMore {
		timeline.BacklogCursor = oldestID
	} else {
		timeline.LatestID = timeline.BacklogStartingPoint
	}
	return results, timeline, hasMore, nil
}
