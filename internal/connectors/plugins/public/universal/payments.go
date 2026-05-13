package universal

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/mappers"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	declared, ok := p.declaredSet()
	if !ok {
		return models.FetchNextPaymentsResponse{}, plugins.ErrNotYetInstalled
	}
	if err := declared.require(models.CAPABILITY_FETCH_PAYMENTS); err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	st, err := decodeState(req.State)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = PAGE_SIZE
	}

	res, err := p.client.ListPayments(ctx, client.Pagination{
		Cursor:        st.NextCursor,
		PageNumber:    st.PageNumber,
		PageSize:      pageSize,
		UpdatedAtFrom: st.LastUpdatedAt,
	})
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	payments := make([]models.PSPPayment, 0, len(res.Items))
	for _, w := range res.Items {
		conv, err := mappers.PaymentToPSPPayment(w)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
		payments = append(payments, conv)
		if w.UpdatedAt.After(st.LastUpdatedAt) {
			st.LastUpdatedAt = w.UpdatedAt
		}
	}

	toDelete := make([]models.PSPPaymentsToDelete, 0, len(res.PaymentsToDelete))
	for _, d := range res.PaymentsToDelete {
		toDelete = append(toDelete, models.PSPPaymentsToDelete{Reference: d.Reference})
	}

	st.NextCursor = res.NextCursor
	if res.NextCursor == "" {
		st.PageNumber++
	}
	newState, err := encodeState(st)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments:         payments,
		PaymentsToDelete: toDelete,
		NewState:         newState,
		HasMore:          res.HasMore,
	}, nil
}
