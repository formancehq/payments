package mangopay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/mangopay/client"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/pagination"
)

type paymentsState struct {
	LastCreationDate time.Time `json:"lastCreationDate"`
	// LastProcessedIDs holds the IDs of all transactions already emitted at exactly
	// LastCreationDate (second precision), accumulated across cycles while the
	// watermark second is unchanged and reset when it advances.
	//
	// Mangopay's AfterDate server filter is EXCLUSIVE second-precision, so we query
	// watermark-1s to re-include the watermark second (M-CON3). Combined with a
	// client-side skip of this set (and a rescan from page 1 each cycle), a
	// same-second group larger than PageSize is walked without a drifting page
	// cursor: every already-emitted sibling is skipped so the scan reaches new rows.
	LastProcessedIDs []string `json:"lastProcessedIDs"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
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
		LastCreationDate: oldState.LastCreationDate,
		LastProcessedIDs: oldState.LastProcessedIDs,
	}

	// Mangopay's AfterDate is an EXCLUSIVE, second-precision server filter. Querying
	// with the raw watermark drops every transaction in the watermark's own second
	// (M-CON3). Subtract one second so that second is returned; fillPayments then
	// skips rows already emitted (LastProcessedIDs) so the rescan reaches new rows.
	afterDate := oldState.LastCreationDate
	if !afterDate.IsZero() {
		afterDate = afterDate.Add(-time.Second)
	}

	var payments []models.PSPPayment
	needMore := false
	hasMore := false
	// Rescan from page 1 each cycle (no drifting page cursor): the processed-ID set
	// skips already-emitted siblings, so the scan always reaches unseen rows.
	for page := 1; ; page++ {
		pagedTransactions, err := p.client.GetTransactions(ctx, from.Reference, page, req.PageSize, afterDate)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		payments, err = fillPayments(pagedTransactions, payments, oldState)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(payments, pagedTransactions, req.PageSize)
		if !needMore || !hasMore {
			break
		}
	}

	// Note that we are NOT trimming data as in other connectors to respect the pageSize; doing so would mean we could
	// lose data as we switch to the next page. We would go over the req.PageSize by max 2x

	if len(payments) > 0 {
		lastCreationDate := payments[len(payments)-1].CreatedAt

		// Collect the references emitted at exactly the new watermark second.
		idsAtWatermark := make([]string, 0)
		for i := range payments {
			if payments[i].CreatedAt.Equal(lastCreationDate) {
				idsAtWatermark = append(idsAtWatermark, payments[i].Reference)
			}
		}

		// Accumulate the processed-ID set while still inside the same watermark
		// second; reset it when the watermark advances to a newer second.
		if lastCreationDate.Equal(oldState.LastCreationDate) {
			newState.LastProcessedIDs = append(oldState.LastProcessedIDs, idsAtWatermark...)
		} else {
			newState.LastProcessedIDs = idsAtWatermark
		}
		newState.LastCreationDate = lastCreationDate
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
	oldState paymentsState,
) ([]models.PSPPayment, error) {
	for _, transaction := range pagedTransactions {
		// Inclusive watermark: skip transactions strictly before it (over-fetched
		// via AfterDate-1s), and any already-emitted transaction at exactly the
		// watermark second. Distinct same-second transactions are kept (M-CON3).
		createdAt := time.Unix(transaction.CreationDate, 0)
		cmp := createdAt.Compare(oldState.LastCreationDate)
		if cmp < 0 || (cmp == 0 && slices.Contains(oldState.LastProcessedIDs, transaction.Id)) {
			continue
		}

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
