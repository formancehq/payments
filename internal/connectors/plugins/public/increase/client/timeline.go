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
	PrevCursor    string   `json:"prev_cursor"`
	LastCursor    string   `json:"last_cursor"`
	AllCursors    []string `json:"all_cursors"`
	BacklogCursor string   `json:"backlog_cursor"`
}

func (t Timeline) IsCaughtUp() bool {
	return t.LastCursor != ""
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
	if timeline.BacklogCursor != "" {
		q.Add("cursor", timeline.BacklogCursor)
	}
	req.URL.RawQuery = q.Encode()

	var res ResponseWrapper[[]*Transaction]
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, timeline, false, fmt.Errorf("failed to get %s for timeline: %w %w", endpoint, err, errRes.Error())
	}

	if len(res.Data) == 0 {
		return nil, timeline, res.NextCursor != "", nil
	}

	// If there's no next cursor, this is our oldest data
	if res.NextCursor == "" {
		hasMore := len(timeline.AllCursors) > 0 || res.NextCursor != ""
		if n := len(timeline.AllCursors); n > 0 {
			timeline.LastCursor = timeline.AllCursors[n-1]
			timeline.AllCursors = timeline.AllCursors[:n-1]
			if n > 1 {
				timeline.PrevCursor = timeline.AllCursors[n-2]
			}
		}
		return res.Data, timeline, hasMore, nil
	}

	// We still have more data to scan
	timeline.BacklogCursor = res.NextCursor
	timeline.AllCursors = append(timeline.AllCursors, res.NextCursor)
	return nil, timeline, true, nil
}
