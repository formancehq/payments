package mangopay

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/mangopay/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type usersState struct {
	LastPage         int       `json:"lastPage"`
	LastCreationDate time.Time `json:"lastCreationDate"`
}

func (p Plugin) fetchNextUsers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {
	var oldState usersState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextOthersResponse{}, err
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

	var users []models.PSPOther
	var userCreationDates []time.Time
	needMore := false
	hasMore := false
	page := oldState.LastPage
	for {
		pagedUsers, err := p.client.GetUsers(ctx, page, req.PageSize)
		if err != nil {
			return models.FetchNextOthersResponse{}, err
		}

		users, userCreationDates, err = fillUsers(pagedUsers, users, userCreationDates, oldState)
		if err != nil {
			return models.FetchNextOthersResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(users, pagedUsers, req.PageSize)
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
		return models.FetchNextOthersResponse{}, err
	}

	return models.FetchNextOthersResponse{
		Others:   users,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func fillUsers(
	pagedUsers []client.User,
	users []models.PSPOther,
	userCreationDates []time.Time,
	oldState usersState,
) ([]models.PSPOther, []time.Time, error) {
	for _, user := range pagedUsers {
		userCreationDate := time.Unix(user.CreationDate, 0)
		switch userCreationDate.Compare(oldState.LastCreationDate) {
		case -1, 0:
			// creationDate <= state.LastCreationDate, nothing to do,
			// we already processed this user.
			continue
		default:
		}

		raw, err := json.Marshal(user)
		if err != nil {
			return nil, nil, err
		}

		users = append(users, models.PSPOther{
			ID:    user.ID,
			Other: raw,
		})
		userCreationDates = append(userCreationDates, userCreationDate)
	}

	return users, userCreationDates, nil
}
