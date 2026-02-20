package column

import (
	"context"
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connectors/column/client"
	"github.com/formancehq/payments/pkg/connector"
)

type paymentsState struct {
	LastIDCreated string          `json:"lastIDCreated"` // deprecated
	Timeline      client.Timeline `json:"timeline"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req connector.FetchNextPaymentsRequest) (connector.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextPaymentsResponse{}, err
		}
	}

	// backwards compatibility for pre PMNT-97
	if oldState.LastIDCreated != "" {
		oldState.Timeline.LastSeenID = oldState.LastIDCreated
	}

	newState := paymentsState{
		Timeline: oldState.Timeline,
	}

	payments := make([]connector.PSPPayment, 0, req.PageSize)
	hasMore := false
	pagedTransactions, timeline, hasMore, err := p.client.GetTransactions(ctx, oldState.Timeline, req.PageSize)
	if err != nil {
		return connector.FetchNextPaymentsResponse{}, err
	}
	newState.Timeline = timeline

	payments, err = p.fillPayments(pagedTransactions, payments, req.PageSize)
	if err != nil {
		return connector.FetchNextPaymentsResponse{}, err
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return connector.FetchNextPaymentsResponse{}, err
	}

	return connector.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func (p *Plugin) fillPayments(
	pagedTransactions []*client.Transaction,
	payments []connector.PSPPayment,
	pageSize int,
) ([]connector.PSPPayment, error) {
	for _, transaction := range pagedTransactions {
		if len(payments) > pageSize {
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

		status := p.mapTransactionStatus(transaction.Status)
		pspPayment := connector.PSPPayment{
			Reference: transaction.ID,
			CreatedAt: createdTime,
			Asset:     *pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, transaction.CurrencyCode)),
			Status:    status,
			Amount:    big.NewInt(transaction.Amount),
			Type:      mapTransactionType(transaction),
			Raw:       raw,
			Metadata: map[string]string{
				client.ColumnUpdatedAtMetadataKey:                         transaction.UpdatedAt,
				client.ColumnCompletedAtMetadataKey:                       transaction.CompletedAt,
				client.ColumnTypeMetadataKey:                              transaction.Type,
				client.ColumnIsIncomingMetadataKey:                        strconv.FormatBool(transaction.IsIncoming),
				client.ColumnIdempotencyKeyMetadataKey:                    transaction.IdempotencyKey,
				client.ColumnDescriptionMetadataKey:                       transaction.Description,
				client.ColumnSenderInternalAccountBankIDMetadataKey:       transaction.SenderInternalAccount.BankAccountID,
				client.ColumnSenderInternalAccountNumberIDMetadataKey:     transaction.SenderInternalAccount.AccountNumberID,
				client.ColumnExternalSourceBankNameMetadataKey:            transaction.ExternalSource.BankName,
				client.ColumnExternalSourceSenderNameMetadataKey:          transaction.ExternalSource.SenderName,
				client.ColumnExternalSourceCounterpartyIDMetadataKey:      transaction.ExternalSource.CounterpartyID,
				client.ColumnReceiverInternalAccountBankIDMetadataKey:     transaction.ReceiverInternalAccount.BankAccountID,
				client.ColumnReceiverInternalAccountNumberIDMetadataKey:   transaction.ReceiverInternalAccount.AccountNumberID,
				client.ColumnExternalDestinationCounterpartyIDMetadataKey: transaction.ExternalDestination.CounterpartyID,
			},
		}
		pspPayment = fillAccountID(transaction, pspPayment)
		payments = append(payments, pspPayment)
	}
	return payments, nil
}

func (p *Plugin) mapTransactionStatus(status string) connector.PaymentStatus {
	status = strings.ToLower(status)
	switch status {
	case "submitted", "pending_submission", "initiated", "pending_deposit", "pending_first_return", "pending_reclear", "pending_return", "pending_second_return", "pending_stop", "pending_user_initiated_return", "scheduled", "pending":
		return connector.PAYMENT_STATUS_PENDING
	case "completed", "deposited", "recleared", "settled", "accepted":
		return connector.PAYMENT_STATUS_SUCCEEDED
	case "canceled", "stopped", "blocked":
		return connector.PAYMENT_STATUS_CANCELLED
	case "failed", "rejected":
		return connector.PAYMENT_STATUS_FAILED
	case "returned", "user_initiated_returned":
		return connector.PAYMENT_STATUS_REFUNDED
	case "return_contested", "return_dishonored", "user_initiated_return_dishonored":
		return connector.PAYMENT_STATUS_REFUND_REVERSED
	case "first_return", "second_return", "user_initiated_return_submitted":
		return connector.PAYMENT_STATUS_REFUNDED_FAILURE
	case "manual_review", "manual_review_approved":
		return connector.PAYMENT_STATUS_AUTHORISATION
	case "hold":
		return connector.PAYMENT_STATUS_CAPTURE
	default:
		return connector.PAYMENT_STATUS_UNKNOWN
	}
}

func fillAccountID(transaction *client.Transaction, pspPayment connector.PSPPayment) connector.PSPPayment {
	assignIfNotEmpty := func(target **string, source *string) {
		if source != nil && *source != "" {
			*target = source
		}
	}

	if transaction.Type == "book" {
		assignIfNotEmpty(&pspPayment.SourceAccountReference, &transaction.SenderInternalAccount.BankAccountID)
		assignIfNotEmpty(&pspPayment.DestinationAccountReference, &transaction.ReceiverInternalAccount.BankAccountID)
	} else if transaction.IsIncoming {
		assignIfNotEmpty(&pspPayment.SourceAccountReference, &transaction.ExternalSource.CounterpartyID)
		assignIfNotEmpty(&pspPayment.DestinationAccountReference, &transaction.ReceiverInternalAccount.BankAccountID)
	} else {
		assignIfNotEmpty(&pspPayment.SourceAccountReference, &transaction.SenderInternalAccount.BankAccountID)
		assignIfNotEmpty(&pspPayment.DestinationAccountReference, &transaction.ExternalDestination.CounterpartyID)
	}

	return pspPayment
}

func mapTransactionType(transaction *client.Transaction) connector.PaymentType {
	if transaction.IsIncoming {
		return connector.PAYMENT_TYPE_PAYIN
	} else if transaction.Type == "book" || transaction.Type == "wire" ||
		transaction.Type == "swift" || transaction.Type == "realtime" ||
		transaction.Type == "check_credit" || transaction.Type == "ach_credit" {
		return connector.PAYMENT_TYPE_PAYOUT
	} else {
		return connector.PAYMENT_TYPE_TRANSFER
	}
}
