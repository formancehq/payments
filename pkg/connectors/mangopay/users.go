package mangopay

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/pkg/connectors/mangopay/client"
	"github.com/formancehq/payments/pkg/connector"
)

type usersState struct {
	LastPage         int       `json:"lastPage"`
	LastCreationDate time.Time `json:"lastCreationDate"`
}

func (p *Plugin) fetchNextUsers(ctx context.Context, req connector.FetchNextOthersRequest) (connector.FetchNextOthersResponse, error) {
	var oldState usersState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextOthersResponse{}, err
		}
	} else {
		oldState = usersState{
			LastPage: 1,
		}
	}

	newState := usersState{
		LastPage:         oldState.LastPage,
		LastCreationDate: oldState.LastCreationDate,
	}

	var users []connector.PSPOther
	var userCreationDates []time.Time
	needMore := false
	hasMore := false
	page := oldState.LastPage
	for {
		pagedUsers, err := p.client.GetUsers(ctx, page, req.PageSize)
		if err != nil {
			return connector.FetchNextOthersResponse{}, err
		}

		users, userCreationDates, err = fillUsers(pagedUsers, users, userCreationDates, oldState)
		if err != nil {
			return connector.FetchNextOthersResponse{}, err
		}

		needMore, hasMore = connector.ShouldFetchMore(users, pagedUsers, req.PageSize)
		if !needMore || !hasMore {
			break
		}

		page++
	}

	if !needMore {
		users = users[:req.PageSize]
		userCreationDates = userCreationDates[:req.PageSize]
	}

	newState.LastPage = page
	if len(userCreationDates) > 0 {
		newState.LastCreationDate = userCreationDates[len(users)-1]
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return connector.FetchNextOthersResponse{}, err
	}

	return connector.FetchNextOthersResponse{
		Others:   users,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func fillUsers(
	pagedUsers []client.User,
	users []connector.PSPOther,
	userCreationDates []time.Time,
	oldState usersState,
) ([]connector.PSPOther, []time.Time, error) {
	for _, user := range pagedUsers {
		userCreationDate := time.Unix(user.CreationDate, 0)
		if userCreationDate.Before(oldState.LastCreationDate) {
			continue
		}

		raw, err := json.Marshal(user)
		if err != nil {
			return nil, nil, err
		}

		users = append(users, connector.PSPOther{
			ID:    user.ID,
			Other: raw,
		})
		userCreationDates = append(userCreationDates, userCreationDate)
	}

	return users, userCreationDates, nil
}
