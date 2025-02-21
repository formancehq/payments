package gocardless

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type usersState struct {
	After            string    `url:"after,omitempty" json:"after,omitempty"`
	Before           string    `url:"before,omitempty" json:"before,omitempty"`
	LastCreationDate time.Time `json:"lastCreationDate"`
}

type UserType struct {
	Reference string `json:"reference"`
}

func (p *Plugin) fetchNextUsers(ctx context.Context, req models.FetchNextOthersRequest) (
	models.FetchNextOthersResponse, error,
) {
	var from UserType
	var oldState usersState

	if req.FromPayload != nil {
		if err := json.Unmarshal(req.FromPayload, &from); err != nil {
			return models.FetchNextOthersResponse{}, err
		}
	}

	if from.Reference == "" {
		return models.FetchNextOthersResponse{}, fmt.Errorf("reference is required")
	}

	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextOthersResponse{}, err
		}
	}

	var users []models.PSPOther
	var userCreationDates []time.Time

	hasMore := false
	needMore := false

	newState := usersState{
		After:            oldState.After,
		Before:           oldState.Before,
		LastCreationDate: oldState.LastCreationDate,
	}

	for {
		if from.Reference[:2] == "CR" {

			pagedCreditors, nextCursor, err := p.client.GetCreditors(ctx, req.PageSize, newState.After, newState.Before)

			if err != nil {
				return models.FetchNextOthersResponse{}, err
			}

			newState.After = nextCursor.After
			newState.Before = nextCursor.Before

			users, userCreationDates, err = fillUsers(pagedCreditors, users, userCreationDates, oldState)

			if err != nil {
				return models.FetchNextOthersResponse{}, err
			}

			needMore, hasMore = pagination.ShouldFetchMore(users, pagedCreditors, req.PageSize)

			if !needMore || !hasMore {
				break
			}

		} else {
			pagedCustomers, nextCursor, err := p.client.GetCustomers(ctx, req.PageSize, newState.After, newState.Before)

			if err != nil {
				return models.FetchNextOthersResponse{}, err
			}

			newState.After = nextCursor.After
			newState.Before = nextCursor.Before

			users, userCreationDates, err = fillUsers(pagedCustomers, users, userCreationDates, oldState)

			if err != nil {
				return models.FetchNextOthersResponse{}, err
			}

			needMore, hasMore = pagination.ShouldFetchMore(users, pagedCustomers, req.PageSize)

			if !needMore || !hasMore {
				break
			}

		}
	}

	if !needMore {
		users = users[:req.PageSize]
		userCreationDates = userCreationDates[:req.PageSize]
	}

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
	pagedUsers []client.GocardlessUser,
	users []models.PSPOther,
	userCreationDates []time.Time,
	oldState usersState,
) ([]models.PSPOther, []time.Time, error) {

	for _, user := range pagedUsers {
		userCreationDate := time.Unix(user.CreatedAt, 0)
		switch userCreationDate.Compare(oldState.LastCreationDate) {
		case -1, 0:
			continue
		default:
		}
		raw, err := json.Marshal(user)

		if err != nil {
			return nil, nil, err
		}

		users = append(users, models.PSPOther{
			ID:    user.Id,
			Other: raw,
		})
		userCreationDates = append(userCreationDates, userCreationDate)

	}

	return users, userCreationDates, nil

}
