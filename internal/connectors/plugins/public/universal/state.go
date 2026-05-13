package universal

import (
	"encoding/json"
	"time"
)

// fetchState is the JSON state struct used by every paginated FetchNext*
// method. It supports both pagination strategies the contract advertises in
// /v1/capabilities's features.pagination:
//
//   - "cursor": opaque NextCursor returned by the counterparty
//   - "page":   1-based PageNumber, incremented locally
//   - "none":   neither — every poll fetches the full set, the engine dedups
//
// LastUpdatedAt is sent on every poll (when set) so the counterparty can
// efficiently filter to records that changed since the last successful run.
// It is not used for pagination — only for incremental fetching — so it can
// be combined with either pagination strategy.
type fetchState struct {
	NextCursor    string    `json:"nextCursor,omitempty"`
	PageNumber    int       `json:"pageNumber,omitempty"`
	LastUpdatedAt time.Time `json:"lastUpdatedAt,omitempty"`
}

func decodeState(raw json.RawMessage) (fetchState, error) {
	var s fetchState
	if len(raw) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, err
	}
	return s, nil
}

func encodeState(s fetchState) (json.RawMessage, error) {
	return json.Marshal(s)
}
