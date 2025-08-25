package checkout

import (
	"context"
	"encoding/json"
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

func mapCheckoutScheme(scheme string) models.PaymentScheme {
	switch strings.ToUpper(scheme) {
		case "VISA":
			return models.PAYMENT_SCHEME_CARD_VISA
		case "MASTERCARD":
			return models.PAYMENT_SCHEME_CARD_MASTERCARD
		case "AMEX", "AMERICAN_EXPRESS":
			return models.PAYMENT_SCHEME_CARD_AMEX
		case "DINERS":
			return models.PAYMENT_SCHEME_CARD_DINERS
		case "DISCOVER":
			return models.PAYMENT_SCHEME_CARD_DISCOVER
		case "JCB":
			return models.PAYMENT_SCHEME_CARD_JCB
		case "UNIONPAY", "UNION_PAY":
			return models.PAYMENT_SCHEME_CARD_UNION_PAY
		case "ALIPAY":
			return models.PAYMENT_SCHEME_CARD_ALIPAY
		case "CUP":
			return models.PAYMENT_SCHEME_CARD_CUP
		case "GOOGLEPAY", "GOOGLE_PAY":
			return models.PAYMENT_SCHEME_GOOGLE_PAY
		case "APPLEPAY", "APPLE_PAY":
			return models.PAYMENT_SCHEME_APPLE_PAY
		case "MAESTRO":
			return models.PAYMENT_SCHEME_MAESTRO
		case "ACH":
			return models.PAYMENT_SCHEME_ACH
		case "ACH_DEBIT":
			return models.PAYMENT_SCHEME_ACH_DEBIT
		case "RTP":
			return models.PAYMENT_SCHEME_RTP
		case "SEPA":
			return models.PAYMENT_SCHEME_SEPA
		case "SEPA_CREDIT":
			return models.PAYMENT_SCHEME_SEPA_CREDIT
		case "SEPA_DEBIT":
			return models.PAYMENT_SCHEME_SEPA_DEBIT
		default:
			return models.PAYMENT_SCHEME_OTHER
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

			paymentType := models.PAYMENT_TYPE_PAYIN
			for _, act := range t.Actions {
				if strings.EqualFold(act.Type, "Payout") {
					paymentType = models.PAYMENT_TYPE_PAYOUT
					break
				}
			}

			payments = append(payments, models.PSPPayment{
				ParentReference: "",
				Reference: t.ID,
				CreatedAt: t.CreatedAt,
				Type: paymentType,
				Amount:    big.NewInt(t.Amount),
				Asset:     asset,
				Scheme: mapCheckoutScheme(t.Scheme),
				Status: mapCheckoutPaymentStatus(t.Status),
				SourceAccountReference: &t.SourceAccountReference,
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

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}
