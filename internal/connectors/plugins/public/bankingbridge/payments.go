package bankingbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/bankingbridge/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState workflowState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	newState := workflowState{
		Cursor: oldState.Cursor,
	}

	payments := make([]models.PSPPayment, 0, req.PageSize)
	pagedTrxs, hasMore, cursor, err := p.client.GetTransactions(ctx, newState.Cursor, req.PageSize)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	for _, trx := range pagedTrxs {
		raw, err := json.Marshal(&trx)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
		payments = append(payments, ToPSPPayment(trx, raw))
	}

	newState.Cursor = cursor
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

func ToPSPPayment(in client.Transaction, raw json.RawMessage) models.PSPPayment {
	amount := big.NewInt(in.AmountInMinors)
	scheme, paymentType := PaymentSchemeAndType(in.BankTransactionCode)
	status := PaymentStatus(in.BankTransactionCode)

	var sourceAccount, destinationAccount *string
	if amount.Sign() < 0 { // negative value means account is being debited
		sourceAccount = pointer.For(in.AccountReference)
		amount.Abs(amount) // convert to a positive amount
	} else {
		destinationAccount = pointer.For(in.AccountReference)
	}

	return models.PSPPayment{
		Reference:                   in.ID,
		CreatedAt:                   in.BookedAt,
		Amount:                      amount,
		Asset:                       in.Asset,
		Scheme:                      scheme,
		Type:                        paymentType,
		Status:                      status,
		SourceAccountReference:      sourceAccount,
		DestinationAccountReference: destinationAccount,
		Metadata: map[string]string{
			"bookingDate":          in.BookingDate,
			"valueDate":            in.ValutaDate,
			"bankTransactionCode":  in.BankTransactionCode,
			"numberofTransactions": fmt.Sprintf("%d", in.NumberOfTransactions),
			"entryReference":       in.EntryReference,
			"servicerReference":    in.ServicerReference,
			"isReversal":           fmt.Sprintf("%t", in.IsReversal),
			"isBatch":              fmt.Sprintf("%t", in.IsBatch),
			"batchMessageId":       in.BatchMessageID,
			"batchPaymentInfoId":   in.BatchPaymentInfoID,
			"importedAt":           in.ImportedAt.String(),
		},
		Raw: raw,
	}
}
