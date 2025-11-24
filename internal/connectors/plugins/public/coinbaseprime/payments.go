package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/coinbaseprime/client"
	"github.com/formancehq/payments/internal/models"
)

type paymentsState struct {
	LastPage int `json:"lastPage"`
}

func mapStatus(s string) models.PaymentStatus {
	s = strings.ToUpper(s)
	switch s {
	case "TRANSACTION_CREATED",
		"TRANSACTION_REQUESTED",
		"TRANSACTION_APPROVED",
		"TRANSACTION_CONSTRUCTED",
		"TRANSACTION_PLANNED",
		"TRANSACTION_PROVISIONED",
		"TRANSACTION_GASSING",
		"TRANSACTION_GASSED",
		"TRANSACTION_BROADCASTING",
		"TRANSACTION_PROCESSING",
		"TRANSACTION_DELAYED",
		"TRANSACTION_RETRIED",
		"TRANSACTION_IMPORT_PENDING":
		return models.PAYMENT_STATUS_PENDING
	case "TRANSACTION_DONE",
		"TRANSACTION_IMPORTED":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "TRANSACTION_CANCELLED":
		return models.PAYMENT_STATUS_CANCELLED
	case "TRANSACTION_REJECTED",
		"TRANSACTION_FAILED":
		return models.PAYMENT_STATUS_FAILED
	case "TRANSACTION_EXPIRED":
		return models.PAYMENT_STATUS_EXPIRED
	case "TRANSACTION_RESTORED",
		"OTHER_TRANSACTION_STATUS":
		return models.PAYMENT_STATUS_OTHER
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

	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextPaymentsResponse{}, models.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	page := oldState.LastPage
	var txs []*client.Transaction
	var err error
	if from.Metadata["spec.coinbase.com/type"] == "wallet" {
		portfolioID := from.Metadata["spec.coinbase.com/portfolio_id"]
		txs, err = p.client.GetWalletTransactions(ctx, portfolioID, from.Reference, page, req.PageSize)
	} else {
		txs, err = p.client.GetPortfolioTransactions(ctx, from.Reference, page, req.PageSize)
	}
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	payments := make([]models.PSPPayment, 0, len(txs))
	for _, t := range txs {
		symbol := strings.ToUpper(t.Symbol)
		precision, ok := supportedCurrenciesWithDecimal[symbol]
		if !ok {
			precision = 8
		}
		amount, err := currency.GetAmountWithPrecisionFromString(t.Amount, precision)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to parse amount: %w", err)
		}
		// Ensure absolute value for the amount
		amount.Abs(amount)

		asset := currency.FormatAsset(supportedCurrenciesWithDecimal, symbol)
		if asset == "" {
			asset = fmt.Sprintf("%s/%d", symbol, precision)
		}

		raw, _ := json.Marshal(t)
		payments = append(payments, models.PSPPayment{
			Reference: t.ID,
			CreatedAt: time.Now().UTC(),
			Type:      models.PAYMENT_TYPE_TRANSFER,
			Amount:    amount,
			Asset:     asset,
			Status:    mapStatus(t.Status),
			Raw:       raw,
		})
	}

	hasMore := len(txs) == req.PageSize
	newState := paymentsState{LastPage: page}
	if hasMore {
		newState.LastPage = page + 1
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
