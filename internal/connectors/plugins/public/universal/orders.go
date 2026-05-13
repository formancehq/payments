package universal

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/mappers"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) FetchNextOrders(ctx context.Context, req models.FetchNextOrdersRequest) (models.FetchNextOrdersResponse, error) {
	res, state, hasMore, err := fetchPaginated(p, ctx, req.State, req.PageSize, models.CAPABILITY_FETCH_ORDERS,
		func(ctx context.Context, page client.Pagination) ([]client.Order, string, bool, error) {
			r, err := p.client.ListOrders(ctx, page)
			if err != nil {
				return nil, "", false, err
			}
			return r.Items, r.NextCursor, r.HasMore, nil
		},
		mappers.OrderToPSPOrder,
	)
	if err != nil {
		return models.FetchNextOrdersResponse{}, err
	}
	return models.FetchNextOrdersResponse{Orders: res, NewState: state, HasMore: hasMore}, nil
}
