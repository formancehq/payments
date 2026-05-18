package universal

import (
	"encoding/json"
	"time"
)

// fetchState is the JSON state shared by every paginated FetchNext*.
// NextCursor/PageNumber cover both contract pagination strategies (cursor
// or 1-based page); LastUpdatedAt is the incremental high-water mark sent
// as `updatedAtFrom` on every poll and advanced by fetchPaginated.
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
