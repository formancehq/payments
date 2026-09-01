package bankingcircle

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/payments/ce/plugins/bankingcircle/client"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/pagination"
)

type paymentsState struct {
	LatestStatusChangedTimestamp time.Time `json:"latestStatusChangedTimestamp"`
	// LatestProcessedIDs holds the PaymentIDs of ALL payments already emitted at
	// exactly LatestStatusChangedTimestamp, accumulated across cycles while the
	// watermark second is unchanged and reset when it advances. BankingCircle has
	// no server-side time filter (we rescan the list from page 1 and filter
	// client-side), so a single boundary ID is not enough: a same-second group
	// larger than PageSize would re-emit earlier siblings and oscillate, never
	// reaching later rows. Tracking the whole set lets the inclusive (>=) filter
	// skip every already-emitted sibling so the scan reaches new rows.
	LatestProcessedIDs []string `json:"latestProcessedIDs"`
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
		LatestProcessedIDs:           oldState.LatestProcessedIDs,
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
		// Trim both slices in lockstep so the watermark and LatestProcessedID are
		// taken from the last *emitted* payment. Trimming only payments would set
		// the watermark from a fetched-but-dropped payment and silently skip the
		// trimmed ones on the next call.
		payments = payments[:req.PageSize]
		latestStatusChangedTimestamps = latestStatusChangedTimestamps[:req.PageSize]
	}

	if len(payments) > 0 {
		lastTimestamp := latestStatusChangedTimestamps[len(latestStatusChangedTimestamps)-1]

		// Collect the references emitted at exactly the new watermark second.
		idsAtWatermark := make([]string, 0)
		for i, ts := range latestStatusChangedTimestamps {
			if ts.Equal(lastTimestamp) {
				idsAtWatermark = append(idsAtWatermark, payments[i].Reference)
			}
		}

		// Accumulate the processed-ID set while still inside the same watermark
		// second; reset it when the watermark advances to a newer second.
		if lastTimestamp.Equal(oldState.LatestStatusChangedTimestamp) {
			newState.LatestProcessedIDs = append(oldState.LatestProcessedIDs, idsAtWatermark...)
		} else {
			newState.LatestProcessedIDs = idsAtWatermark
		}
		newState.LatestStatusChangedTimestamp = lastTimestamp
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
		// Inclusive watermark: skip payments strictly before the watermark, and
		// the single already-processed payment at exactly the watermark. Distinct
		// payments sharing that timestamp are kept (M-CON2: equal-timestamp rows
		// were previously dropped at page/cycle boundaries).
		cmp := payment.LatestStatusChangedTimestamp.Compare(oldState.LatestStatusChangedTimestamp)
		if cmp < 0 || (cmp == 0 && slices.Contains(oldState.LatestProcessedIDs, payment.PaymentID)) {
			continue
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
