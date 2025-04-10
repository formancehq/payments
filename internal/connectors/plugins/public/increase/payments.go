package increase

import (
	"context"
	"encoding/json"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

type paymentsState struct {
	NextSucceededCursor string `json:"next_succeeded_cursor"`
	NextPendingCursor   string `json:"next_pending_cursor"`
	NextDeclinedCursor  string `json:"next_declined_cursor"`
	StopSucceeded       bool   `json:"stop_succeeded"`
	StopPending         bool   `json:"stop_pending"`
	StopDeclined        bool   `json:"stop_declined"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	newState := paymentsState{
		NextSucceededCursor: oldState.NextSucceededCursor,
		NextPendingCursor:   oldState.NextPendingCursor,
		NextDeclinedCursor:  oldState.NextDeclinedCursor,
		StopSucceeded:       oldState.StopSucceeded,
		StopPending:         oldState.StopPending,
		StopDeclined:        oldState.StopDeclined,
	}

	payments := make([]models.PSPPayment, 0, req.PageSize)
	payments, hasMore, err := p.processPaymentTypes(ctx, &newState, payments, req.PageSize)
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

func (p *Plugin) processPaymentTypes(ctx context.Context, state *paymentsState, payments []models.PSPPayment, pageSize int) ([]models.PSPPayment, bool, error) {
	var err error
	payments, err = p.processPendingPayments(ctx, state, payments, pageSize)
	if err != nil {
		return nil, false, err
	}

	payments, err = p.processSucceededPayments(ctx, state, payments, pageSize)
	if err != nil {
		return nil, false, err
	}

	payments, err = p.processDeclinedPayments(ctx, state, payments, pageSize)
	if err != nil {
		return nil, false, err
	}

	hasMore := !(state.StopPending && state.StopSucceeded && state.StopDeclined)

	return payments, hasMore, nil
}

func (p *Plugin) fillPayments(
	pagedTransactions []*client.Transaction,
	payments []models.PSPPayment,
	pageSize int,
	status models.PaymentStatus,
) ([]models.PSPPayment, error) {
	for i, transaction := range pagedTransactions {
		if i > pageSize*3 {
			break
		}

		createdTime, err := time.Parse(time.RFC3339, transaction.CreatedAt)
		if err != nil {
			return nil, err
		}

		raw, err := json.Marshal(transaction)
		if err != nil {
			return nil, err
		}

		pspPayment := models.PSPPayment{
			Reference: transaction.ID,
			CreatedAt: createdTime,
			Asset:     *pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, transaction.Currency)),
			Status:    status,
			Amount:    big.NewInt(int64(math.Abs(float64(transaction.Amount)))),
			Type:      mapTransactionType(transaction.Source.Category),
			Raw:       raw,
			Metadata: map[string]string{
				client.IncreaseRouteIDMetadataKey:        transaction.RouteID,
				client.IncreaseRouteTypeMetadataKey:      transaction.RouteType,
				client.IncreaseSourceCategoryMetadataKey: transaction.Source.Category,
			},
		}
		pspPayment = fillAccountID(transaction, pspPayment)
		payments = append(payments, pspPayment)
	}

	return payments, nil
}

func (p *Plugin) processPendingPayments(ctx context.Context, state *paymentsState, payments []models.PSPPayment, pageSize int) ([]models.PSPPayment, error) {
	if state.StopPending {
		return payments, nil
	}

	pagedPendingTransactions, nextPendingCursor, err := p.client.GetPendingTransactions(ctx, pageSize, state.NextPendingCursor)
	if err != nil {
		return nil, err
	}

	payments, err = p.fillPayments(pagedPendingTransactions, payments, pageSize, models.PAYMENT_STATUS_PENDING)
	if err != nil {
		return nil, err
	}

	state.NextPendingCursor = nextPendingCursor

	state.StopPending = nextPendingCursor == ""

	return payments, nil
}

func (p *Plugin) processSucceededPayments(ctx context.Context, state *paymentsState, payments []models.PSPPayment, pageSize int) ([]models.PSPPayment, error) {
	if state.StopSucceeded {
		return payments, nil
	}

	pagedTransactions, nextSucceededCursor, err := p.client.GetTransactions(ctx, pageSize, state.NextSucceededCursor)
	if err != nil {
		return nil, err
	}

	payments, err = p.fillPayments(pagedTransactions, payments, pageSize, models.PAYMENT_STATUS_SUCCEEDED)
	if err != nil {
		return nil, err
	}

	state.NextSucceededCursor = nextSucceededCursor

	state.StopSucceeded = nextSucceededCursor == ""

	return payments, nil
}

func (p *Plugin) processDeclinedPayments(ctx context.Context, state *paymentsState, payments []models.PSPPayment, pageSize int) ([]models.PSPPayment, error) {
	if state.StopDeclined {
		return payments, nil
	}

	pagedDeclinedTransactions, nextDeclinedCursor, err := p.client.GetDeclinedTransactions(ctx, pageSize, state.NextDeclinedCursor)
	if err != nil {
		return nil, err
	}

	payments, err = p.fillPayments(pagedDeclinedTransactions, payments, pageSize, models.PAYMENT_STATUS_FAILED)
	if err != nil {
		return nil, err
	}

	state.NextDeclinedCursor = nextDeclinedCursor

	state.StopDeclined = nextDeclinedCursor == ""

	return payments, nil
}

func (p *Plugin) mapPayment(transaction *client.Transaction, status models.PaymentStatus) (models.PSPPayment, error) {
	createdTime, err := time.Parse(time.RFC3339, transaction.CreatedAt)
	if err != nil {
		return models.PSPPayment{}, err
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
		Amount:    big.NewInt(int64(math.Abs(float64(transaction.Amount)))),
		Type:      mapTransactionType(transaction.Source.Category),
		Raw:       raw,
		Metadata: map[string]string{
			client.IncreaseRouteIDMetadataKey:        transaction.RouteID,
			client.IncreaseRouteTypeMetadataKey:      transaction.RouteType,
			client.IncreaseSourceCategoryMetadataKey: transaction.Source.Category,
		},
	}

	pspPayment = fillAccountID(transaction, pspPayment)

	return pspPayment, nil
}

func fillAccountID(transaction *client.Transaction, pspPayment models.PSPPayment) models.PSPPayment {
	category := transaction.Source.Category
	if (category == "account_transfer_intention" && transaction.Amount > 0) || isPayin(category) {
		pspPayment.DestinationAccountReference = &transaction.AccountID
	} else {
		pspPayment.SourceAccountReference = &transaction.AccountID
	}

	return pspPayment
}

func isPayin(transactionType string) bool {
	if strings.HasPrefix(transactionType, "inbound_") ||
		transactionType == "ach_transfer_return" ||
		transactionType == "card_refund" ||
		transactionType == "interest_payment" ||
		transactionType == "check_deposit_return" ||
		transactionType == "check_transfer_deposit" ||
		transactionType == "cashback_payment" ||
		transactionType == "check_deposit_instruction" ||
		transactionType == "check_deposit_acceptance" {
		return true
	}
	return false
}

func isPayout(transactionType string) bool {
	if transactionType == "wire_transfer_intention" ||
		transactionType == "real_time_payments_transfer_acknowledgement" ||
		transactionType == "ach_transfer_intention" ||
		transactionType == "fee_payment" ||
		transactionType == "ach_transfer_instruction" ||
		transactionType == "check_transfer_instruction" ||
		transactionType == "wire_transfer_instruction" ||
		transactionType == "real_time_payments_transfer_instruction" {
		return true
	}
	return false
}

func mapTransactionType(transactionType string) models.PaymentType {
	if isPayin(transactionType) {
		return models.PAYMENT_TYPE_PAYIN
	} else if isPayout(transactionType) {
		return models.PAYMENT_TYPE_PAYOUT
	} else if transactionType == "account_transfer_intention" ||
		transactionType == "account_transfer_instruction" {
		return models.PAYMENT_TYPE_TRANSFER
	} else {
		return models.PAYMENT_TYPE_OTHER
	}
}
