package routable

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/routable/client"
	"github.com/formancehq/payments/internal/models"
)

type paymentsState struct {
	NextPage int `json:"nextPage"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	state := paymentsState{}
	if len(req.State) != 0 {
		_ = json.Unmarshal(req.State, &state)
	}

	page := state.NextPage
	if page == 0 {
		page = 1
	}

	var accountID string
	if len(req.FromPayload) != 0 {
		var acc models.PSPAccount
		_ = json.Unmarshal(req.FromPayload, &acc)
		accountID = acc.Reference
	}

	var transactions []*client.Transaction
	var err error
	if accountID != "" {
		transactions, err = p.client.GetTransactionsByAccount(ctx, page, req.PageSize, accountID)
	} else {
		transactions, err = p.client.GetTransactions(ctx, page, req.PageSize)
	}
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	payments := make([]models.PSPPayment, 0, len(transactions))
	for _, t := range transactions {
		createdAt, _ := time.Parse(time.RFC3339, t.CreatedAt)
		precision, _ := currency.GetPrecision(supportedCurrenciesWithDecimal, t.CurrencyCode)
		amount, _ := currency.GetAmountWithPrecisionFromString(func() string {
			if t.Amount == "" {
				return "0"
			}
			return t.Amount
		}(), precision)
		asset := currency.FormatAsset(supportedCurrenciesWithDecimal, func() string {
			if t.CurrencyCode == "" {
				return "USD"
			}
			return t.CurrencyCode
		}())

		status := models.PAYMENT_STATUS_PENDING
		switch t.Status {
		case "completed":
			status = models.PAYMENT_STATUS_SUCCEEDED
		case "failed", "canceled":
			status = models.PAYMENT_STATUS_FAILED
		default:
			status = models.PAYMENT_STATUS_PENDING
		}

		ptype := models.PAYMENT_TYPE_PAYOUT
		if t.Type == "external" {
			ptype = models.PAYMENT_TYPE_TRANSFER
		}

		raw, _ := json.Marshal(t)
		pay := models.PSPPayment{
			ParentReference:             "",
			Reference:                   t.ID,
			CreatedAt:                   createdAt,
			Type:                        ptype,
			Amount:                      amount,
			Asset:                       asset,
			Scheme:                      models.PAYMENT_SCHEME_OTHER,
			Status:                      status,
			SourceAccountReference:      nil,
			DestinationAccountReference: nil,
			Metadata:                    map[string]string{"spec.formance.com/generic_provider": ProviderName},
			Raw:                         raw,
		}
		if t.WithdrawFromAccount.ID != "" {
			pay.SourceAccountReference = &t.WithdrawFromAccount.ID
		}
		payments = append(payments, pay)
	}

	// Simplified paging: advance page if we returned a full page
	hasMore := len(transactions) == req.PageSize
	newState := paymentsState{NextPage: page + 1}
	payload, _ := json.Marshal(newState)
	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}
