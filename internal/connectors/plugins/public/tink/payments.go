package tink

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
)

type paymentsState struct {
	NextPageToken string `json:"nextPageToken"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	newState := paymentsState{
		NextPageToken: oldState.NextPageToken,
	}

	var from models.OpenBankingForwardedUserFromPayload
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	var webhook fetchNextDataRequest
	if err := json.Unmarshal(from.FromPayload, &webhook); err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	payments := make([]models.PSPPayment, 0, req.PageSize)
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
			return models.FetchNextPaymentsResponse{}, err
		}

		payments, err = toPSPPayments(payments, resp.Transactions, from)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
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
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func toPSPPayments(
	payments []models.PSPPayment,
	transactions []client.Transaction,
	from models.OpenBankingForwardedUserFromPayload,
) ([]models.PSPPayment, error) {
	for _, transaction := range transactions {
		amount, asset, err := MapTinkAmount(transaction.Amount.Value.Value, transaction.Amount.Value.Scale, transaction.Amount.CurrencyCode)
		if err != nil {
			return payments, err
		}

		var sourceReference *string
		var destinationReference *string
		var paymentType models.PaymentType
		if amount.Sign() < 0 {
			paymentType = models.PAYMENT_TYPE_PAYOUT
			sourceReference = &transaction.AccountID
		} else {
			paymentType = models.PAYMENT_TYPE_PAYIN
			destinationReference = &transaction.AccountID
		}

		amount = new(big.Int).Abs(amount)

		raw, err := json.Marshal(transaction)
		if err != nil {
			return payments, err
		}

		var status models.PaymentStatus
		switch transaction.Status {
		case "BOOKED":
			status = models.PAYMENT_STATUS_SUCCEEDED
		case "PENDING":
			status = models.PAYMENT_STATUS_PENDING
		default:
			status = models.PAYMENT_STATUS_OTHER
		}

		p := models.PSPPayment{
			Reference:                   transaction.ID,
			CreatedAt:                   transaction.TransactionDateTime,
			Type:                        paymentType,
			Amount:                      amount,
			Asset:                       *asset,
			Scheme:                      models.PAYMENT_SCHEME_OTHER,
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
