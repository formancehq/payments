package gocardless

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless/client"
	"github.com/formancehq/payments/internal/models"
)

type usersState struct {
	CustomersAfter string `url:"customersAfter,omitempty" json:"customersAfter,omitempty"`

	CreditorsAfter string `url:"creditorsAfter,omitempty" json:"creditorsAfter,omitempty"`
}

func (p *Plugin) fetchNextUsers(ctx context.Context, req models.FetchNextOthersRequest) (
	models.FetchNextOthersResponse, error,
) {
	var oldState usersState

	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextOthersResponse{}, err
		}
	}

	var users []models.PSPOther
	hasMore := false

	newState := usersState{
		CustomersAfter: oldState.CustomersAfter,

		CreditorsAfter: oldState.CreditorsAfter,
	}

	creditorsUsers, creditorsHasMore, creditorsState, err := p.getCreditorsUsers(ctx, req.PageSize, newState)

	if err != nil {
		return models.FetchNextOthersResponse{}, err
	}

	customersUsers, customersHasMore, customersState, err := p.getCustomersUsers(ctx, req.PageSize, newState)

	if err != nil {
		return models.FetchNextOthersResponse{}, err
	}

	hasMore = *creditorsHasMore || *customersHasMore

	customerAfter := customersState.CustomersAfter
	if customerAfter == "" {
		customerAfter = newState.CustomersAfter
	}

	creditorAfter := creditorsState.CreditorsAfter
	if creditorAfter == "" {
		creditorAfter = newState.CreditorsAfter
	}

	newState = usersState{
		CustomersAfter: customerAfter,

		CreditorsAfter: creditorAfter,
	}

	users = append(users, customersUsers...)
	users = append(users, creditorsUsers...)

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

func (p *Plugin) getCreditorsUsers(
	ctx context.Context,
	pageSize int,
	newState usersState,
) ([]models.PSPOther, *bool, *usersState, error) {

	hasMore := false

	var users []models.PSPOther

	pagedCreditors, nextCursor, err := p.client.GetCreditors(ctx, pageSize, newState.CreditorsAfter)

	if err != nil {
		return []models.PSPOther{}, nil, nil, err
	}

	newState.CreditorsAfter = nextCursor.After

	users, err = fillUsers(pagedCreditors, users)

	if err != nil {
		return []models.PSPOther{}, nil, nil, err

	}

	hasMore = nextCursor.After != ""

	if !hasMore && len(users) > 0 {
		newState.CreditorsAfter = users[len(users)-1].ID
	}

	if len(users) > pageSize {
		users = users[:pageSize]
	}

	return users, &hasMore, &newState, nil
}

func (p *Plugin) getCustomersUsers(
	ctx context.Context,
	pageSize int,
	newState usersState,
) ([]models.PSPOther, *bool, *usersState, error) {

	hasMore := false
	var users []models.PSPOther

	pagedCustomers, nextCursor, err := p.client.GetCustomers(ctx, pageSize, newState.CustomersAfter)

	if err != nil {
		return []models.PSPOther{}, nil, nil, err
	}

	newState.CustomersAfter = nextCursor.After

	users, err = fillUsers(pagedCustomers, users)

	if err != nil {
		return []models.PSPOther{}, nil, nil, err
	}

	hasMore = nextCursor.After != ""

	if !hasMore && len(users) > 0 {
		newState.CustomersAfter = users[len(users)-1].ID
	}

	if len(users) > pageSize {
		users = users[:pageSize]
	}

	return users, &hasMore, &newState, nil
}

func fillUsers(
	pagedUsers []client.GocardlessUser,
	users []models.PSPOther,
) ([]models.PSPOther, error) {

	for _, user := range pagedUsers {

		raw, err := json.Marshal(user)

		if err != nil {
			return nil, err
		}

		users = append(users, models.PSPOther{
			ID:    user.Id,
			Other: raw,
		})

	}

	return users, nil

}
