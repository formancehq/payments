package universal

import (
	"context"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/mappers"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/pkg/errors"
)

func (p *Plugin) FetchNextOthers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return models.FetchNextOthersResponse{}, errorsutils.NewWrappedError(errors.New("FetchNextOthers requires a non-empty req.Name"), models.ErrInvalidRequest)
	}
	res, state, hasMore, err := fetchPaginated(p, ctx, req.State, req.PageSize, models.CAPABILITY_FETCH_OTHERS,
		func(ctx context.Context, page client.Pagination) ([]client.Other, string, bool, error) {
			r, err := p.client.ListOthers(ctx, name, page)
			if err != nil {
				return nil, "", false, err
			}
			return r.Items, r.NextCursor, r.HasMore, nil
		},
		mappers.OtherToPSPOther,
		// client.Other has no wire timestamp — fixed `time.Time{}`
		// keeps LastUpdatedAt frozen, which is the contract (others
		// are opaque to incremental fetch).
		func(_ client.Other) time.Time { return time.Time{} },
	)
	if err != nil {
		return models.FetchNextOthersResponse{}, err
	}
	return models.FetchNextOthersResponse{Others: res, NewState: state, HasMore: hasMore}, nil
}
