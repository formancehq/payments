package increase

import (
	"context"
	"encoding/json"
	"math"
	"math/big"
	"sort"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

type paymentsState struct {
	StopSucceeded     bool            `json:"stop_succeeded"`
	StopPending       bool            `json:"stop_pending"`
	StopDeclined      bool            `json:"stop_declined"`
	SucceededTimeline client.Timeline `json:"succeeded_timeline"`
	DeclinedTimeline  client.Timeline `json:"declined_timeline"`
	PendingTimeline   client.Timeline `json:"pending_timeline"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	newState := paymentsState{
		StopSucceeded:     oldState.StopSucceeded,
		StopPending:       oldState.StopPending,
		StopDeclined:      oldState.StopDeclined,
		SucceededTimeline: oldState.SucceededTimeline,
		PendingTimeline:   oldState.PendingTimeline,
		DeclinedTimeline:  oldState.DeclinedTimeline,
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

	sort.SliceStable(payments, func(i, j int) bool {
		return payments[i].CreatedAt.Before(payments[j].CreatedAt)
	})

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

		pspPayment, err := p.mapPayment(transaction, status)
		if err != nil {
			return nil, err
		}

		payments = append(payments, pspPayment)
	}

	return payments, nil
}

func getTransferID(transaction *client.Transaction) string {
	if transaction.Source.TransferID != "" {
		return transaction.Source.TransferID
	}
	if transaction.Source.WireTransferID != "" {
		return transaction.Source.WireTransferID
	}
	if transaction.Source.CheckDepositID != "" {
		return transaction.Source.CheckDepositID
	}
	if transaction.Source.InboundCheckDepositID != "" {
		return transaction.Source.InboundCheckDepositID
	}
	if transaction.Source.InboundAchTransferID != "" {
		return transaction.Source.InboundAchTransferID
	}
	if transaction.Source.InboundWireTransferID != "" {
		return transaction.Source.InboundWireTransferID
	}
	if transaction.Source.ID != "" {
		return transaction.Source.ID
	}
	return ""
}

func (p *Plugin) processPendingPayments(ctx context.Context, state *paymentsState, payments []models.PSPPayment, pageSize int) ([]models.PSPPayment, error) {
	if state.StopPending {
		return payments, nil
	}

	pagedPendingTransactions, timeline, hasMore, err := p.client.GetPendingTransactions(ctx, pageSize, state.PendingTimeline)
	if err != nil {
		return nil, err
	}

	payments, err = p.fillPayments(pagedPendingTransactions, payments, pageSize, models.PAYMENT_STATUS_PENDING)
	if err != nil {
		return nil, err
	}

	state.PendingTimeline = timeline
	state.StopPending = !hasMore

	return payments, nil
}

func (p *Plugin) processSucceededPayments(ctx context.Context, state *paymentsState, payments []models.PSPPayment, pageSize int) ([]models.PSPPayment, error) {
	if state.StopSucceeded {
		return payments, nil
	}

	pagedTransactions, timeline, hasMore, err := p.client.GetTransactions(ctx, pageSize, state.SucceededTimeline)
	if err != nil {
		return nil, err
	}

	payments, err = p.fillPayments(pagedTransactions, payments, pageSize, models.PAYMENT_STATUS_SUCCEEDED)
	if err != nil {
		return nil, err
	}

	state.SucceededTimeline = timeline
	state.StopSucceeded = !hasMore

	return payments, nil
}

func (p *Plugin) processDeclinedPayments(ctx context.Context, state *paymentsState, payments []models.PSPPayment, pageSize int) ([]models.PSPPayment, error) {
	if state.StopDeclined {
		return payments, nil
	}

	pagedDeclinedTransactions, timeline, hasMore, err := p.client.GetDeclinedTransactions(ctx, pageSize, state.DeclinedTimeline)
	if err != nil {
		return nil, err
	}

	payments, err = p.fillPayments(pagedDeclinedTransactions, payments, pageSize, models.PAYMENT_STATUS_FAILED)
	if err != nil {
		return nil, err
	}

	state.DeclinedTimeline = timeline
	state.StopDeclined = !hasMore

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

	paymentStatus := mapTransactionStatus(transaction.Source.Category, status)

	pspPayment := models.PSPPayment{
		Reference: transaction.ID,
		CreatedAt: createdTime,
		Asset:     *pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, transaction.Currency)),
		Status:    paymentStatus,
		Amount:    big.NewInt(int64(math.Abs(float64(transaction.Amount)))),
		Type:      mapTransactionType(transaction),
		Raw:       raw,
		Metadata: map[string]string{
			client.IncreaseRouteIDMetadataKey:        transaction.RouteID,
			client.IncreaseRouteTypeMetadataKey:      transaction.RouteType,
			client.IncreaseSourceCategoryMetadataKey: transaction.Source.Category,
		},
	}
	pspPayment = fillParentReference(transaction, pspPayment)
	pspPayment = fillAccountID(transaction, pspPayment)

	return pspPayment, nil
}

func fillParentReference(transaction *client.Transaction, pspPayment models.PSPPayment) models.PSPPayment {
	transferID := getTransferID(transaction)
	if transferID != "" {
		pspPayment.ParentReference = transferID
	}

	return pspPayment
}

func fillAccountID(transaction *client.Transaction, pspPayment models.PSPPayment) models.PSPPayment {
	category := transaction.Source.Category
	transactionAmount := transaction.Amount
	if category == "account_transfer_intention" && transactionAmount < 0 {
		pspPayment.SourceAccountReference = &transaction.Source.SourceAccountID
		pspPayment.DestinationAccountReference = &transaction.Source.DestinationAccountID
	} else if category == "account_transfer_intention" && transactionAmount > 0 {
		pspPayment.SourceAccountReference = &transaction.Source.DestinationAccountID
		pspPayment.DestinationAccountReference = &transaction.Source.SourceAccountID
	} else if isPayin(category) {
		pspPayment.DestinationAccountReference = &transaction.AccountID
	} else {
		pspPayment.SourceAccountReference = &transaction.AccountID
	}

	return pspPayment
}

func isPayin(transactionType string) bool {
	if transactionType == "inbound_ach_transfer" ||
		transactionType == "inbound_ach_transfer_return_intention" ||
		transactionType == "inbound_funds_hold" ||
		transactionType == "inbound_check_adjustment" ||
		transactionType == "inbound_check_deposit_return_intention" ||
		transactionType == "inbound_real_time_payments_transfer_confirmation" ||
		transactionType == "inbound_real_time_payments_transfer_decline" ||
		transactionType == "inbound_wire_transfer" ||
		transactionType == "inbound_wire_transfer_reversal" ||
		transactionType == "card_refund" ||
		transactionType == "interest_payment" ||
		transactionType == "check_transfer_deposit" ||
		transactionType == "cashback_payment" ||
		transactionType == "check_deposit_instruction" ||
		transactionType == "check_deposit_rejection" ||
		transactionType == "check_decline" ||
		transactionType == "wire_decline" ||
		transactionType == "ach_decline" ||
		transactionType == "check_deposit_return" ||
		transactionType == "check_deposit_acceptance" {
		return true
	}
	return false
}

func isPayout(transactionType string) bool {
	if transactionType == "wire_transfer_intention" ||
		transactionType == "real_time_payments_transfer_acknowledgement" ||
		transactionType == "ach_transfer_intention" ||
		transactionType == "ach_transfer_return" ||
		transactionType == "fee_payment" ||
		transactionType == "inbound_wire_reversal" ||
		transactionType == "ach_transfer_instruction" ||
		transactionType == "check_transfer_instruction" ||
		transactionType == "wire_transfer_instruction" ||
		transactionType == "real_time_payments_transfer_instruction" {
		return true
	}
	return false
}

func mapTransactionType(transaction *client.Transaction) models.PaymentType {
	transactionType := transaction.Source.Category
	transactionAmount := transaction.Amount
	if isPayin(transactionType) ||
		(transactionType == "account_transfer_intention" && transactionAmount > 0) {
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

func mapTransactionStatus(transactionType string, status models.PaymentStatus) models.PaymentStatus {
	switch transactionType {
	case "ach_transfer_return", "inbound_ach_transfer_return_intention",
		"check_deposit_return", "inbound_check_deposit_return_intention",
		"inbound_wire_reversal", "inbound_wire_transfer_reversal":
		return models.PAYMENT_STATUS_REFUNDED
	default:
		return status
	}
}
