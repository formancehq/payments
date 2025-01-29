package bankingcircle

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/bankingcircle/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type paymentsState struct {
	LatestStatusChangedTimestamp time.Time `json:"latestStatusChangedTimestamp"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	newState := paymentsState{
		LatestStatusChangedTimestamp: oldState.LatestStatusChangedTimestamp,
	}

	var payments []models.PSPPayment
	var latestStatusChangedTimestamps []time.Time
	needMore := false
	hasMore := false
	for page := 1; ; page++ {
		pagedPayments, err := p.client.GetPayments(ctx, page, req.PageSize)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		payments, latestStatusChangedTimestamps, err = fillPayments(pagedPayments, payments, latestStatusChangedTimestamps, oldState)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(payments, pagedPayments, req.PageSize)
		if !needMore || !hasMore {
			break
		}
	}

	if !needMore {
		payments = payments[:req.PageSize]
	}

	if len(payments) > 0 {
		newState.LatestStatusChangedTimestamp = latestStatusChangedTimestamps[len(latestStatusChangedTimestamps)-1]
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
	pagedPayments []client.Payment,
	payments []models.PSPPayment,
	latestStatusChangedTimestamps []time.Time,
	oldState paymentsState,
) ([]models.PSPPayment, []time.Time, error) {
	for _, payment := range pagedPayments {
		switch payment.LatestStatusChangedTimestamp.Compare(oldState.LatestStatusChangedTimestamp) {
		case -1, 0:
			continue
		default:
		}

		p, err := translatePayment(payment)
		if err != nil {
			return nil, nil, err
		}

		if p != nil {
			payments = append(payments, *p)
			latestStatusChangedTimestamps = append(latestStatusChangedTimestamps, payment.LatestStatusChangedTimestamp)
		}
	}

	return payments, latestStatusChangedTimestamps, nil
}

func translatePayment(from client.Payment) (*models.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return nil, err
	}

	paymentType := matchPaymentType(from.Classification)

	pCurrency := from.Transfer.Amount.Currency
	if pCurrency == "" {
		// If payment is pending, then we need to use the debtor information
		// to get currency and amount.
		pCurrency = from.DebtorInformation.DebitAmount.Currency
	}

	pAmount := from.Transfer.Amount.Amount.String()
	if pAmount == "" {
		// If payment is pending, then we need to use the debtor information
		// to get currency and amount.
		pAmount = from.DebtorInformation.DebitAmount.Amount.String()
	}

	precision, ok := supportedCurrenciesWithDecimal[pCurrency]
	if !ok {
		return nil, nil
	}

	amount, err := currency.GetAmountWithPrecisionFromString(pAmount, precision)
	if err != nil {
		return nil, err
	}

	createdAt := from.ProcessedTimestamp
	if createdAt.IsZero() {
		createdAt = from.LastChangedTimestamp
	}

	payment := models.PSPPayment{
		Reference: from.PaymentID,
		CreatedAt: createdAt,
		Type:      paymentType,
		Amount:    amount,
		Asset:     currency.FormatAsset(supportedCurrenciesWithDecimal, pCurrency),
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    matchPaymentStatus(from.Status),
		Raw:       raw,
	}

	if from.DebtorInformation.AccountID != "" {
		payment.SourceAccountReference = &from.DebtorInformation.AccountID
	}

	if from.CreditorInformation.AccountID != "" {
		payment.DestinationAccountReference = &from.CreditorInformation.AccountID
	}

	return &payment, nil
}

func matchPaymentStatus(paymentStatus string) models.PaymentStatus {
	switch paymentStatus {
	case "Processed":
		return models.PAYMENT_STATUS_SUCCEEDED
	// On MissingFunding - the payment is still in progress.
	// If there will be funds available within 10 days - the payment will be processed.
	// Otherwise - it will be cancelled.
	case "PendingProcessing", "MissingFunding":
		return models.PAYMENT_STATUS_PENDING
	case "Rejected", "Cancelled", "Reversed", "Returned":
		return models.PAYMENT_STATUS_FAILED
	}

	return models.PAYMENT_STATUS_OTHER
}

func matchPaymentType(paymentType string) models.PaymentType {
	switch paymentType {
	case "Incoming":
		return models.PAYMENT_TYPE_PAYIN
	case "Outgoing":
		return models.PAYMENT_TYPE_PAYOUT
	case "Own":
		return models.PAYMENT_TYPE_TRANSFER
	}

	return models.PAYMENT_TYPE_OTHER
}
