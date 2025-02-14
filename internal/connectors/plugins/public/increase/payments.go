package increase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

type paymentsState struct {
	LastSucceededCreatedAt time.Time `json:"last_succeeded_created_at"`
	LastPendingCreatedAt   time.Time `json:"last_pending_created_at"`
	LastDeclinedCreatedAt  time.Time `json:"last_declined_created_at"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var (
		oldState paymentsState
		newState paymentsState
	)
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	payments := make([]models.PSPPayment, 0, req.PageSize)
	payments, hasMore, err := p.processPaymentTypes(ctx, oldState, payments, req.PageSize/3, &newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
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

func (p *Plugin) processPaymentTypes(ctx context.Context, oldState paymentsState, payments []models.PSPPayment, pageSize int, newState *paymentsState) ([]models.PSPPayment, bool, error) {
	payments, pendingCursor, err := p.processPendingPayments(ctx, oldState, payments, pageSize, newState)
	if err != nil {
		return nil, false, err
	}

	payments, succeededCursor, err := p.processSucceededPayments(ctx, oldState, payments, pageSize, newState)
	if err != nil {
		return nil, false, err
	}

	payments, declinedCursor, err := p.processDeclinedPayments(ctx, oldState, payments, pageSize, newState)
	if err != nil {
		return nil, false, err
	}

	hasMore := pendingCursor != "" || succeededCursor != "" || declinedCursor != ""
	return payments, hasMore, nil
}

func (p *Plugin) fillPayments(
	pagedTransactions []*client.Transaction,
	payments []models.PSPPayment,
	pageSize int,
	status models.PaymentStatus,
) ([]models.PSPPayment, error) {
	for i, transaction := range pagedTransactions {
		if i >= pageSize {
			break
		}

		precision, ok := supportedCurrenciesWithDecimal[transaction.Currency]
		if !ok {
			return nil, nil
		}

		amount, err := currency.GetAmountWithPrecisionFromString(transaction.Amount.String(), precision)
		if err != nil {
			return nil, fmt.Errorf("failed to parse amount %s: %w", transaction.Amount, err)
		}

		createdTime, err := time.Parse(time.RFC3339, transaction.CreatedAt)
		if err != nil {
			return nil, err
		}

		raw, err := json.Marshal(transaction)
		if err != nil {
			return nil, err
		}

		payments = append(payments, models.PSPPayment{
			Reference: transaction.ID,
			CreatedAt: createdTime,
			Asset:     *pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, transaction.Currency)),
			Status:    status,
			Amount:    amount,
			Type:      models.PAYMENT_TYPE_OTHER,
			Raw:       raw,
		})
	}

	return payments, nil
}

func (p *Plugin) processPendingPayments(ctx context.Context, oldState paymentsState, payments []models.PSPPayment, pageSize int, newState *paymentsState) ([]models.PSPPayment, string, error) {
	pagedPendingTransactions, nextPendingCursor, err := p.client.GetPendingTransactions(ctx, pageSize, oldState.LastPendingCreatedAt)
	if err != nil {
		return nil, "", err
	}

	payments, err = p.fillPayments(pagedPendingTransactions, payments, pageSize, models.PAYMENT_STATUS_PENDING)
	if err != nil {
		return nil, "", err
	}

	if len(payments) > 0 && payments[len(payments)-1].Status == models.PAYMENT_STATUS_PENDING {
		newState.LastPendingCreatedAt = payments[len(payments)-1].CreatedAt
	}

	return payments, nextPendingCursor, nil
}

func (p *Plugin) processSucceededPayments(ctx context.Context, oldState paymentsState, payments []models.PSPPayment, pageSize int, newState *paymentsState) ([]models.PSPPayment, string, error) {
	pagedTransactions, nextSucceededCursor, err := p.client.GetTransactions(ctx, pageSize, oldState.LastSucceededCreatedAt)
	if err != nil {
		return nil, "", err
	}

	payments, err = p.fillPayments(pagedTransactions, payments, pageSize, models.PAYMENT_STATUS_SUCCEEDED)
	if err != nil {
		return nil, "", err
	}

	if len(payments) > 0 && payments[len(payments)-1].Status == models.PAYMENT_STATUS_SUCCEEDED {
		newState.LastSucceededCreatedAt = payments[len(payments)-1].CreatedAt
	}

	return payments, nextSucceededCursor, nil
}

func (p *Plugin) processDeclinedPayments(ctx context.Context, oldState paymentsState, payments []models.PSPPayment, pageSize int, newState *paymentsState) ([]models.PSPPayment, string, error) {
	pagedDeclinedTransactions, nextDeclinedCursor, err := p.client.GetDeclinedTransactions(ctx, pageSize, oldState.LastDeclinedCreatedAt)
	if err != nil {
		return nil, "", err
	}

	payments, err = p.fillPayments(pagedDeclinedTransactions, payments, pageSize, models.PAYMENT_STATUS_FAILED)
	if err != nil {
		return nil, "", err
	}

	if len(payments) > 0 && payments[len(payments)-1].Status == models.PAYMENT_STATUS_FAILED {
		newState.LastDeclinedCreatedAt = payments[len(payments)-1].CreatedAt
	}

	return payments, nextDeclinedCursor, nil
}

func (p *Plugin) mapPayment(transaction *client.Transaction, status models.PaymentStatus) (models.PSPPayment, error) {
	createdTime, err := time.Parse(time.RFC3339, transaction.CreatedAt)
	if err != nil {
		return models.PSPPayment{}, err
	}

	precision, ok := supportedCurrenciesWithDecimal[transaction.Currency]
	if !ok {
		return models.PSPPayment{}, nil
	}

	amount, err := currency.GetAmountWithPrecisionFromString(transaction.Amount.String(), precision)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to parse amount %s: %w", transaction.Amount, err)
	}

	raw, err := json.Marshal(transaction)
	if err != nil {
		return models.PSPPayment{}, err
	}

	pspPayment := models.PSPPayment{
		Reference: transaction.ID,
		CreatedAt: createdTime,
		Asset:     *pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, transaction.Currency)),
		Status:    status,
		Amount:    amount,
		Type:      models.PAYMENT_TYPE_OTHER,
		Raw:       raw,
	}

	return pspPayment, nil
}
