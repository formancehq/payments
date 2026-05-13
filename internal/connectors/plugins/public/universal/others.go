package universal

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/mappers"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) FetchNextOthers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {
	declared, ok := p.declaredSet()
	if !ok {
		return models.FetchNextOthersResponse{}, plugins.ErrNotYetInstalled
	}
	if err := declared.require(models.CAPABILITY_FETCH_OTHERS); err != nil {
		return models.FetchNextOthersResponse{}, err
	}

	st, err := decodeState(req.State)
	if err != nil {
		return models.FetchNextOthersResponse{}, err
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = PAGE_SIZE
	}

	res, err := p.client.ListOthers(ctx, req.Name, client.Pagination{
		Cursor:        st.NextCursor,
		PageNumber:    st.PageNumber,
		PageSize:      pageSize,
		UpdatedAtFrom: st.LastUpdatedAt,
	})
	if err != nil {
		return models.FetchNextOthersResponse{}, err
	}

	out := make([]models.PSPOther, 0, len(res.Items))
	for _, w := range res.Items {
		o, err := mappers.OtherToPSPOther(w)
		if err != nil {
			return models.FetchNextOthersResponse{}, err
		}
		out = append(out, o)
	}

	st.NextCursor = res.NextCursor
	if res.NextCursor == "" {
		st.PageNumber++
	}
	newState, err := encodeState(st)
	if err != nil {
		return models.FetchNextOthersResponse{}, err
	}

	return models.FetchNextOthersResponse{Others: out, NewState: newState, HasMore: res.HasMore}, nil
}
