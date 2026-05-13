package universal

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/mappers"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	res, state, hasMore, err := fetchPaginated(p, ctx, req.State, req.PageSize, models.CAPABILITY_FETCH_ACCOUNTS,
		func(ctx context.Context, page client.Pagination) ([]client.Account, string, bool, error) {
			r, err := p.client.ListAccounts(ctx, page)
			if err != nil {
				return nil, "", false, err
			}
			return r.Items, r.NextCursor, r.HasMore, nil
		},
		mappers.AccountToPSPAccount,
	)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}
	return models.FetchNextAccountsResponse{Accounts: res, NewState: state, HasMore: hasMore}, nil
}

// fetchPaginated centralises every FetchNext* concern: capability guard,
// "not installed" check, pagination state encode/decode, and per-item
// translation. Each per-primitive file stays a thin wrapper that picks
// the list endpoint and the wire→PSP translator from the mappers
// package.
//
// Cursor-based and page-based pagination are both honoured: if the
// counterparty returns a NextCursor we use it; otherwise we increment a
// PageNumber locally. The engine's pageSize is honoured (0 → PAGE_SIZE).
func fetchPaginated[Wire any, PSP any](
	p *Plugin,
	ctx context.Context,
	rawState []byte,
	pageSize int,
	capability models.Capability,
	list func(context.Context, client.Pagination) ([]Wire, string, bool, error),
	convert func(Wire) (PSP, error),
) ([]PSP, []byte, bool, error) {
	declared, ok := p.declaredSet()
	if !ok {
		return nil, nil, false, plugins.ErrNotYetInstalled
	}
	if err := declared.require(capability); err != nil {
		return nil, nil, false, err
	}

	st, err := decodeState(rawState)
	if err != nil {
		return nil, nil, false, err
	}
	if pageSize <= 0 {
		pageSize = PAGE_SIZE
	}

	items, nextCursor, hasMore, err := list(ctx, client.Pagination{
		Cursor:        st.NextCursor,
		PageNumber:    st.PageNumber,
		PageSize:      pageSize,
		UpdatedAtFrom: st.LastUpdatedAt,
	})
	if err != nil {
		return nil, nil, false, err
	}

	converted := make([]PSP, 0, len(items))
	for _, w := range items {
		c, err := convert(w)
		if err != nil {
			return nil, nil, false, err
		}
		converted = append(converted, c)
	}

	st.NextCursor = nextCursor
	if nextCursor == "" {
		st.PageNumber++
	}
	newState, err := encodeState(st)
	if err != nil {
		return nil, nil, false, err
	}
	return converted, newState, hasMore, nil
}
