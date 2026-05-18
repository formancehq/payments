package universal

import (
	"context"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/mappers"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) FetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	res, state, hasMore, err := fetchPaginated(p, ctx, req.State, req.PageSize, models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
		func(ctx context.Context, page client.Pagination) ([]client.Account, string, bool, error) {
			r, err := p.client.ListExternalAccounts(ctx, page)
			if err != nil {
				return nil, "", false, err
			}
			return r.Items, r.NextCursor, r.HasMore, nil
		},
		mappers.AccountToPSPAccount,
		func(a client.Account) time.Time { return a.CreatedAt },
	)
	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
	}
	return models.FetchNextExternalAccountsResponse{ExternalAccounts: res, NewState: state, HasMore: hasMore}, nil
}
