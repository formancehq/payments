package moov

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

type usersState struct {
	Skip int64 `url:"skip,omitempty" json:"skip,omitempty"`
}

func (p *Plugin) fetchNextUsers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {
	var oldState usersState

	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextOthersResponse{}, fmt.Errorf("failed to unmarshal from payload: %w", err)
		}
	}

	var users []models.PSPOther
	hasMore := false

	newState := usersState{
		Skip: oldState.Skip,
	}

	moovUsers, err := p.client.GetUsers(ctx, int(oldState.Skip), req.PageSize)
	if err != nil {
		return models.FetchNextOthersResponse{}, err
	}

	hasMore = len(moovUsers) == int(req.PageSize)

	newState.Skip = int64(len(moovUsers)) + oldState.Skip

	for _, user := range moovUsers {
		otherUser, err := json.Marshal(user)

		if err != nil {
			return models.FetchNextOthersResponse{}, fmt.Errorf("failed to marshal user: %w", err)
		}

		users = append(users, models.PSPOther{
			ID:    user.AccountID,
			Other: otherUser,
		})
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextOthersResponse{}, fmt.Errorf("failed to marshal new state: %w", err)
	}

	return models.FetchNextOthersResponse{
		Others:   users,
		HasMore:  hasMore,
		NewState: payload,
	}, nil
}
