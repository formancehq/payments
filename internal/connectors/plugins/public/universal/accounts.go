package universal

import (
	"context"
	"time"

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
		func(a client.Account) time.Time { return a.CreatedAt },
	)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}
	return models.FetchNextAccountsResponse{Accounts: res, NewState: state, HasMore: hasMore}, nil
}

// fetchPaginated centralises every FetchNext* concern: capability
// guard, install check, pagination state, per-item translation and
// high-water `updatedAtFrom` advancement.
//
// `timestampOf` lets the helper advance LastUpdatedAt for every
// primitive (without it the next poll re-fetches the whole window —
// see Coinbase #707). The watermark is computed in a local and only
// committed to the returned state once the whole batch has converted
// successfully so a partial failure can never advance past unseen rows.
func fetchPaginated[Wire any, PSP any](
	p *Plugin,
	ctx context.Context,
	rawState []byte,
	pageSize int,
	capability models.Capability,
	list func(context.Context, client.Pagination) ([]Wire, string, bool, error),
	convert func(Wire) (PSP, error),
	timestampOf func(Wire) time.Time,
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
	highWater := st.LastUpdatedAt
	for _, w := range items {
		c, err := convert(w)
		if err != nil {
			return nil, nil, false, err
		}
		converted = append(converted, c)
		if timestampOf != nil {
			if t := timestampOf(w); t.After(highWater) {
				highWater = t
			}
		}
	}

	st.LastUpdatedAt = highWater
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
