package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// Timeline tracks the pagination state for fetching transactions
type Timeline struct {
	// Cursors encountered while scanning backwards to find oldest records
	// These will be used in reverse order to fetch forward chronologically
	Cursors []string `json:"cursors"`
	// Whether we've found the oldest records
	FoundOldest bool `json:"found_oldest"`
}

func (t Timeline) IsCaughtUp() bool {
	return t.FoundOldest
}

func (c *client) scanForOldest(
	ctx context.Context,
	timeline Timeline,
	endpoint string,
	pageSize int,
) ([]*Transaction, Timeline, bool, error) {
	req, err := c.newRequest(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, timeline, false, fmt.Errorf("failed to create timeline transactions request: %w", err)
	}

	q := req.URL.Query()
	q.Add("limit", strconv.Itoa(pageSize))

	// Use the last cursor if we have any
	if len(timeline.Cursors) > 0 {
		q.Add("cursor", timeline.Cursors[len(timeline.Cursors)-1])
	}
	req.URL.RawQuery = q.Encode()

	var res ResponseWrapper[[]*Transaction]
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, timeline, false, fmt.Errorf("failed to get %s for timeline: %w %w", endpoint, err, errRes.Error())
	}

	// If we got no data, we're done scanning
	if len(res.Data) == 0 {
		timeline.FoundOldest = true
		return nil, timeline, false, nil
	}

	// If there's no next cursor, we've found the oldest data
	if res.NextCursor == "" {
		timeline.FoundOldest = true
		if len(timeline.Cursors) > 0 {
			timeline.Cursors = timeline.Cursors[:len(timeline.Cursors)-1]
		}
		return res.Data, timeline, len(timeline.Cursors) > 0, nil
	}

	// Store the next cursor and continue scanning
	timeline.Cursors = append(timeline.Cursors, res.NextCursor)
	return nil, timeline, true, nil
}
