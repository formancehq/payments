package universal

import (
	"context"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/mappers"
	"github.com/formancehq/payments/internal/models"
)

// FetchNextPayments is the only fetch primitive that surfaces deletions —
// the helper handles the common shape, the local closure captures
// res.PaymentsToDelete out-of-band so we can return it alongside.
func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var toDelete []models.PSPPaymentsToDelete
	payments, state, hasMore, err := fetchPaginated(p, ctx, req.State, req.PageSize, models.CAPABILITY_FETCH_PAYMENTS,
		func(ctx context.Context, page client.Pagination) ([]client.Payment, string, bool, error) {
			r, err := p.client.ListPayments(ctx, page)
			if err != nil {
				return nil, "", false, err
			}
			for _, d := range r.PaymentsToDelete {
				toDelete = append(toDelete, models.PSPPaymentsToDelete{Reference: d.Reference})
			}
			return r.Items, r.NextCursor, r.HasMore, nil
		},
		mappers.PaymentToPSPPayment,
		func(w client.Payment) time.Time { return w.UpdatedAt },
	)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}
	return models.FetchNextPaymentsResponse{
		Payments:         payments,
		PaymentsToDelete: toDelete,
		NewState:         state,
		HasMore:          hasMore,
	}, nil
}
