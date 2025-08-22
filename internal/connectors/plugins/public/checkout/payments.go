package checkout

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
	"github.com/formancehq/go-libs/v3/currency"
)

type paymentsState struct {
	LastPage int `json:"lastPage"`
}

func mapCheckoutPaymentStatus(s string) models.PaymentStatus {
	if s == "" {
		return models.PAYMENT_STATUS_UNKNOWN
	}
	ls := strings.ToLower(strings.TrimSpace(s))

	switch ls {
		case "authorized", "authorised", "card verified", "approved":
			return models.PAYMENT_STATUS_AUTHORISATION
		case "captured", "capture", "partially captured":
			return models.PAYMENT_STATUS_CAPTURE
		case "refunded", "partially refunded":
			return models.PAYMENT_STATUS_REFUNDED
		case "pending", "capture pending", "refund pending":
			return models.PAYMENT_STATUS_PENDING
		case "declined", "failed", "failure":
			return models.PAYMENT_STATUS_FAILED
		case "expired":
			return models.PAYMENT_STATUS_EXPIRED
		case "canceled", "cancelled", "voided", "void":
			return models.PAYMENT_STATUS_CANCELLED
		case "refund declined", "refund_failed", "refund failed":
			return models.PAYMENT_STATUS_REFUNDED_FAILURE
		case "refund reversed", "reversed":
			return models.PAYMENT_STATUS_REFUND_REVERSED
		case "disputed", "chargeback":
			return models.PAYMENT_STATUS_DISPUTE
		case "chargeback won", "dispute won":
			return models.PAYMENT_STATUS_DISPUTE_WON
		case "chargeback lost", "dispute lost":
			return models.PAYMENT_STATUS_DISPUTE_LOST
		default:
			return models.PAYMENT_STATUS_OTHER
	}
}


func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	startPage := oldState.LastPage + 1
	newState := paymentsState{
		LastPage: oldState.LastPage,
	}

	payments := make([]models.PSPPayment, 0, req.PageSize)
	needMore := false
	hasMore := false

	for page := startPage; ; page++ {
		pagedTxs, err := p.client.GetTransactions(ctx, page, req.PageSize)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		for _, t := range pagedTxs {
			raw, _ := json.Marshal(t)

			asset := currency.FormatAsset(supportedCurrenciesWithDecimal, t.Currency)

			md := map[string]string{
				"payment_id": t.PaymentID,
				"type":       t.Type,
				"status":     t.Status,
			}

			payments = append(payments, models.PSPPayment{
				ParentReference: "",
				Reference: t.ID,
				Amount:    big.NewInt(t.Amount),
				Asset:     asset,
				CreatedAt: t.CreatedAt,
				SourceAccountReference: &t.SourceAccountReference,
				Status: mapCheckoutPaymentStatus(t.Status),
				Scheme: models.PAYMENT_SCHEME_CARD_ALIPAY,
				Type: models.PAYMENT_TYPE_UNKNOWN,
				Metadata:  md,
				Raw:       raw,
			})
		}

		needMore, hasMore = pagination.ShouldFetchMore(payments, pagedTxs, req.PageSize)
		newState.LastPage = page

		if !needMore || !hasMore {
			break
		}
	}

	if !needMore && len(payments) > req.PageSize {
		payments = payments[:req.PageSize]
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	if t, _ := json.Marshal(payments); true {
		fmt.Printf("[checkout] payments returns %d payment(s): %s\n", len(payments), string(t))
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}
