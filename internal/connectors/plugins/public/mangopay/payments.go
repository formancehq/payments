package mangopay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/mangopay/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type paymentsState struct {
	LastPage         int       `json:"lastPage"`
	LastCreationDate time.Time `json:"lastCreationDate"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	} else {
		oldState = paymentsState{
			LastPage: 1,
		}
	}

	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextPaymentsResponse{}, errors.New("missing from payload when fetching payments")
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	newState := paymentsState{
		LastPage:         oldState.LastPage,
		LastCreationDate: oldState.LastCreationDate,
	}

	var payments []models.PSPPayment
	needMore := false
	hasMore := false
	page := oldState.LastPage
	for {
		pagedTransactions, err := p.client.GetTransactions(ctx, from.Reference, page, req.PageSize, oldState.LastCreationDate)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		payments, err = fillPayments(pagedTransactions, payments)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(payments, pagedTransactions, req.PageSize)
		if !needMore || !hasMore {
			break
		}

		page++
	}

	// Note that we are NOT trimming data as in other connectors to respect the pageSize; doing so would mean we could
	// lose data as we switch to the next page. We would go over the req.PageSize by max 2x
	//if !needMore {
	//	payments = payments[:req.PageSize]
	//}

	if len(payments) > 0 {
		if oldState.LastCreationDate.Equal(payments[len(payments)-1].CreatedAt) {
			newState.LastPage = page + 1
		} else {
			newState.LastPage = 1
		}
		newState.LastCreationDate = payments[len(payments)-1].CreatedAt

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

func fillPayments(
	pagedTransactions []client.Payment,
	payments []models.PSPPayment,
) ([]models.PSPPayment, error) {
	for _, transaction := range pagedTransactions {
		payment, err := transactionToPayment(transaction)
		if err != nil {
			return nil, err
		}

		if payment != nil {
			payments = append(payments, *payment)
		}
	}
	return payments, nil
}

func transactionToPayment(from client.Payment) (*models.PSPPayment, error) {
	raw, err := json.Marshal(&from)
	if err != nil {
		return nil, err
	}

	paymentType := matchPaymentType(from.Type)
	paymentStatus := matchPaymentStatus(from.Status)

	var amount big.Int
	_, ok := amount.SetString(from.DebitedFunds.Amount.String(), 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse amount %s", from.DebitedFunds.Amount.String())
	}

	payment := models.PSPPayment{
		Reference: from.Id,
		CreatedAt: time.Unix(from.CreationDate, 0),
		Type:      paymentType,
		Amount:    &amount,
		Asset:     currency.FormatAsset(supportedCurrenciesWithDecimal, from.DebitedFunds.Currency),
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    paymentStatus,
		Raw:       raw,
	}

	if from.DebitedWalletID != "" {
		payment.SourceAccountReference = &from.DebitedWalletID
	}

	if from.CreditedWalletID != "" {
		payment.DestinationAccountReference = &from.CreditedWalletID
	}

	return &payment, nil
}

func matchPaymentType(paymentType string) models.PaymentType {
	switch paymentType {
	case "PAYIN":
		return models.PAYMENT_TYPE_PAYIN
	case "PAYOUT":
		return models.PAYMENT_TYPE_PAYOUT
	case "TRANSFER":
		return models.PAYMENT_TYPE_TRANSFER
	}

	return models.PAYMENT_TYPE_OTHER
}

func matchPaymentStatus(paymentStatus string) models.PaymentStatus {
	switch paymentStatus {
	case "CREATED":
		return models.PAYMENT_STATUS_PENDING
	case "SUCCEEDED":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "FAILED":
		return models.PAYMENT_STATUS_FAILED
	}

	return models.PAYMENT_STATUS_OTHER
}
