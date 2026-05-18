package universal

import (
	"context"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/mappers"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) FetchNextConversions(ctx context.Context, req models.FetchNextConversionsRequest) (models.FetchNextConversionsResponse, error) {
	res, state, hasMore, err := fetchPaginated(p, ctx, req.State, req.PageSize, models.CAPABILITY_FETCH_CONVERSIONS,
		func(ctx context.Context, page client.Pagination) ([]client.Conversion, string, bool, error) {
			r, err := p.client.ListConversions(ctx, page)
			if err != nil {
				return nil, "", false, err
			}
			return r.Items, r.NextCursor, r.HasMore, nil
		},
		mappers.ConversionToPSPConversion,
		func(c client.Conversion) time.Time { return c.CreatedAt },
	)
	if err != nil {
		return models.FetchNextConversionsResponse{}, err
	}
	return models.FetchNextConversionsResponse{Conversions: res, NewState: state, HasMore: hasMore}, nil
}
