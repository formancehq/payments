package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// Timeline allows the client to navigate the backlog and decide whether to fetch
// historical or recently added data
type Timeline struct {
	LatestID      string `json:"latest_id"`
	BacklogCursor string `json:"backlog_cursor"`
}

func (t Timeline) IsCaughtUp() bool {
	return t.LatestID != ""
}

func (c *client) scanForOldest(
	ctx context.Context,
	timeline Timeline,
	endpoint string,
	pageSize int,
) (*Transaction, Timeline, bool, error) {
	req, err := c.newRequest(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, timeline, false, fmt.Errorf("failed to create timeline transactions request: %w", err)
	}

	q := req.URL.Query()
	q.Add("limit", strconv.Itoa(pageSize))
	if timeline.BacklogCursor != "" {
		q.Add("starting_after", timeline.BacklogCursor)
	}
	req.URL.RawQuery = q.Encode()

	var res TransactionResponseWrapper[[]*Transaction]
	var errRes columnError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, timeline, false, fmt.Errorf("failed to get transactions for timeline: %w %w", err, errRes.Error())
	}

	if len(res.Transfers) == 0 {
		return nil, timeline, res.HasMore, nil
	}

	oldest := res.Transfers[len(res.Transfers)-1]
	if !res.HasMore {
		timeline.LatestID = oldest.ID
		return oldest, timeline, res.HasMore, nil
	}
	timeline.BacklogCursor = oldest.ID
	return oldest, timeline, res.HasMore, nil
}
