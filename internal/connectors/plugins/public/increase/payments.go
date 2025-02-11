package increase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

type paymentsState struct {
	NextSucceededCursor string `json:"next_succeeded_cursor"`
	NextPendingCursor   string `json:"next_pending_cursor"`
	NextDeclinedCursor  string `json:"next_declined_cursor"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	payments := make([]models.PSPPayment, 0, req.PageSize)
	hasMore := false
	pagedTransactions, nextSucceededCursor, err := p.client.GetTransactions(ctx, req.PageSize, oldState.NextSucceededCursor)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	payments, err = p.fillPayments(pagedTransactions, payments, req.PageSize, models.PAYMENT_STATUS_SUCCEEDED)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	pagedPendingTransactions, nextPendingCursor, err := p.client.GetPendingTransactions(ctx, req.PageSize, oldState.NextPendingCursor)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	payments, err = p.fillPayments(pagedPendingTransactions, payments, req.PageSize, models.PAYMENT_STATUS_PENDING)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	pagedDeclinedTransactions, nextDeclinedCursor, err := p.client.GetDeclinedTransactions(ctx, req.PageSize, oldState.NextDeclinedCursor)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	payments, err = p.fillPayments(pagedDeclinedTransactions, payments, req.PageSize, models.PAYMENT_STATUS_FAILED)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	hasMore = nextSucceededCursor != "" || nextDeclinedCursor != "" || nextPendingCursor != ""

	newState := paymentsState{
		NextSucceededCursor: nextSucceededCursor,
		NextPendingCursor:   nextPendingCursor,
		NextDeclinedCursor:  nextDeclinedCursor,
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func (p *Plugin) fillPayments(
	pagedTransactions []*client.Transaction,
	payments []models.PSPPayment,
	pageSize int,
	status models.PaymentStatus,
) ([]models.PSPPayment, error) {
	for _, transaction := range pagedTransactions {
		if len(payments) >= pageSize {
			break
		}

		createdTime, err := time.Parse("2006-01-02T15:04:05.999-0700", transaction.CreatedAt)
		if err != nil {
			return nil, err
		}

		raw, err := json.Marshal(transaction)
		if err != nil {
			return nil, err
		}

		payments = append(payments, models.PSPPayment{
			Reference:                   transaction.ID,
			CreatedAt:                   createdTime,
			Asset:                       *pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, transaction.Currency)),
			SourceAccountReference:      &transaction.Source.SourceAccountID,
			DestinationAccountReference: &transaction.Source.DestinationAccountID,
			Status:                      status,
			Raw:                         raw,
		})
	}

	return payments, nil
}
