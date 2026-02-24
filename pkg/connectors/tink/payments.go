package tink

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/payments/pkg/connectors/tink/client"
	"github.com/formancehq/payments/pkg/connector"
)

type paymentsState struct {
	NextPageToken string `json:"nextPageToken"`
}

func computeCreatedAt(transaction client.Transaction) time.Time {
	if !transaction.TransactionDateTime.IsZero() {
		return transaction.TransactionDateTime
	}
	if !transaction.Dates.Transaction.IsZero() {
		return transaction.Dates.Transaction
	}
	if !transaction.BookedDateTime.IsZero() {
		return transaction.BookedDateTime
	}
	if !transaction.Dates.Booked.IsZero() {
		return transaction.Dates.Booked
	}
	return time.Now()
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req connector.FetchNextPaymentsRequest) (connector.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextPaymentsResponse{}, err
		}
	}

	newState := paymentsState{
		NextPageToken: oldState.NextPageToken,
	}

	var from connector.OpenBankingForwardedUserFromPayload
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return connector.FetchNextPaymentsResponse{}, err
	}

	var webhook fetchNextDataRequest
	if err := json.Unmarshal(from.FromPayload, &webhook); err != nil {
		return connector.FetchNextPaymentsResponse{}, err
	}

	payments := make([]connector.PSPPayment, 0, req.PageSize)
	hasMore := false
	for {
		resp, err := p.client.ListTransactions(ctx, client.ListTransactionRequest{
			UserID:        webhook.ExternalUserID,
			AccountID:     webhook.AccountID,
			BookedDateGTE: webhook.TransactionEarliestModifiedBookedDate,
			BookedDateLTE: webhook.TransactionLatestModifiedBookedDate,
			PageSize:      req.PageSize,
			NextPageToken: newState.NextPageToken,
		})
		if err != nil {
			return connector.FetchNextPaymentsResponse{}, err
		}

		payments, err = toPSPPayments(payments, resp.Transactions, from)
		if err != nil {
			return connector.FetchNextPaymentsResponse{}, err
		}

		newState.NextPageToken = resp.NextPageToken
		hasMore = resp.NextPageToken != ""
		if resp.NextPageToken == "" {
			break
		}

		needMore := len(payments) < req.PageSize

		if !needMore || !hasMore {
			break
		}
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

func toPSPPayments(
	payments []connector.PSPPayment,
	transactions []client.Transaction,
	from connector.OpenBankingForwardedUserFromPayload,
) ([]connector.PSPPayment, error) {
	for _, transaction := range transactions {
		amount, asset, err := MapTinkAmount(transaction.Amount.Value.Value, transaction.Amount.Value.Scale, transaction.Amount.CurrencyCode)
		if err != nil {
			return payments, err
		}

		var sourceReference *string
		var destinationReference *string
		var paymentType connector.PaymentType
		if amount.Sign() < 0 {
			paymentType = connector.PAYMENT_TYPE_PAYOUT
			sourceReference = &transaction.AccountID
		} else {
			paymentType = connector.PAYMENT_TYPE_PAYIN
			destinationReference = &transaction.AccountID
		}

		amount = new(big.Int).Abs(amount)

		raw, err := json.Marshal(transaction)
		if err != nil {
			return payments, err
		}

		var status connector.PaymentStatus
		switch transaction.Status {
		case "BOOKED":
			status = connector.PAYMENT_STATUS_SUCCEEDED
		case "PENDING":
			status = connector.PAYMENT_STATUS_PENDING
		default:
			status = connector.PAYMENT_STATUS_OTHER
		}

		p := connector.PSPPayment{
			Reference:                   transaction.ID,
			CreatedAt:                   computeCreatedAt(transaction),
			Type:                        paymentType,
			Amount:                      amount,
			Asset:                       *asset,
			Scheme:                      connector.PAYMENT_SCHEME_OTHER,
			Status:                      status,
			SourceAccountReference:      sourceReference,
			DestinationAccountReference: destinationReference,
			PsuID:                       &from.PSUID,
			Metadata:                    make(map[string]string),
			Raw:                         raw,
		}

		if from.OpenBankingConnection != nil {
			p.OpenBankingConnectionID = &from.OpenBankingConnection.ConnectionID
		}

		payments = append(payments, p)
	}

	return payments, nil
}
