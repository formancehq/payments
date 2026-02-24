package wise

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/formancehq/payments/pkg/connector"
)

type profilesState struct {
	// Profiles are ordered by their ID
	LastProfileID uint64 `json:"lastProfileID"`
}

func (p *Plugin) fetchNextProfiles(ctx context.Context, req connector.FetchNextOthersRequest) (connector.FetchNextOthersResponse, error) {
	var oldState profilesState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextOthersResponse{}, err
		}
	}

	newState := profilesState{
		LastProfileID: oldState.LastProfileID,
	}

	var others []connector.PSPOther
	hasMore := false
	profiles, err := p.client.GetProfiles(ctx)
	if err != nil {
		return connector.FetchNextOthersResponse{}, err
	}

	for _, profile := range profiles {
		if profile.ID <= oldState.LastProfileID {
			continue
		}

		raw, err := json.Marshal(profile)
		if err != nil {
			return connector.FetchNextOthersResponse{}, err
		}

		others = append(others, connector.PSPOther{
			ID:    strconv.FormatUint(profile.ID, 10),
			Other: raw,
		})

		newState.LastProfileID = profile.ID

		if len(others) >= req.PageSize {
			hasMore = true
			break
		}
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return connector.FetchNextOthersResponse{}, err
	}

	return connector.FetchNextOthersResponse{
		Others:   others,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}
