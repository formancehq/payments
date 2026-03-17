package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

type paymentsState struct {
	Offset int `json:"offset"`
}

const bitstampDatetimeLayout = "2006-01-02 15:04:05.000000"

// Bitstamp user_transactions type values.
const (
	txTypeDeposit             = "0"
	txTypeWithdrawal          = "1"
	txTypeMarketTrade         = "2"
	txTypeSubAccountTransfer  = "14"
	txTypeStakingCredit       = "25"
	txTypeStakingSent         = "26"
	txTypeStakingReward       = "27"
	txTypeReferralReward      = "32"
	txTypeSettlementTransfer  = "33"
	txTypeInterAccountTransfer = "35"
)

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	transactions, err := p.client.GetUserTransactions(ctx, oldState.Offset, req.PageSize)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	payments := make([]models.PSPPayment, 0, len(transactions))
	for _, tx := range transactions {
		payment, err := p.transactionToPayment(tx)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to convert transaction %d: %w", tx.ID, err)
		}
		if payment == nil {
			continue
		}
		payments = append(payments, *payment)
	}

	newState := paymentsState{Offset: oldState.Offset + len(transactions)}
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  len(transactions) == req.PageSize,
	}, nil
}

func (p *Plugin) transactionToPayment(tx client.UserTransaction) (*models.PSPPayment, error) {
	asset, precision, ok := p.resolveAssetAndPrecision(tx.CurrencyAmounts)
	if !ok {
		p.logger.Infof("skipping transaction %d: no matching currency found", tx.ID)
		return nil, nil
	}

	raw, err := json.Marshal(tx)
	if err != nil {
		return nil, err
	}

	paymentType := transactionTypeToPaymentType(tx.Type)

	// All Bitstamp user_transactions are completed — the API only returns
	// settled transactions, unlike Coinbase Prime which returns all states.
	status := models.PAYMENT_STATUS_SUCCEEDED

	amount, err := p.extractAmount(tx.CurrencyAmounts, asset, precision)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount: %w", err)
	}

	createdAt, err := time.Parse(bitstampDatetimeLayout, tx.Datetime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse datetime %q: %w", tx.Datetime, err)
	}

	metadata := buildTransactionMetadata(tx)

	payment := models.PSPPayment{
		Reference: strconv.FormatInt(tx.ID, 10),
		CreatedAt: createdAt,
		Type:      paymentType,
		Amount:    amount,
		Asset:     asset,
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    status,
		Metadata:  metadata,
		Raw:       raw,
	}

	return &payment, nil
}

// resolveAssetAndPrecision finds the primary currency in the transaction
// using a deterministic strategy: pick the currency with the largest absolute
// value among known currencies. This avoids non-determinism from Go map
// iteration order on multi-currency transactions (trades).
func (p *Plugin) resolveAssetAndPrecision(amounts map[string]string) (string, int, bool) {
	var bestSymbol string
	var bestPrecision int
	var bestAbsVal float64
	found := false

	for key, val := range amounts {
		symbol := strings.ToUpper(strings.TrimSpace(key))
		precision, ok := p.currencies[symbol]
		if !ok {
			continue
		}

		// Parse absolute value for comparison.
		cleanVal := strings.TrimPrefix(val, "-")
		fval, _, err := new(big.Float).Parse(cleanVal, 10)
		if err != nil {
			continue
		}
		absVal, _ := fval.Float64()

		// Skip zero amounts.
		if absVal == 0 {
			continue
		}

		if !found || absVal > bestAbsVal {
			bestSymbol = symbol
			bestPrecision = precision
			bestAbsVal = absVal
			found = true
		}
	}

	if !found {
		return "", 0, false
	}

	asset := currency.FormatAsset(p.currencies, bestSymbol)
	return asset, bestPrecision, true
}

func (p *Plugin) extractAmount(amounts map[string]string, targetAsset string, precision int) (*big.Int, error) {
	for key, val := range amounts {
		symbol := strings.ToUpper(strings.TrimSpace(key))
		if _, ok := p.currencies[symbol]; !ok {
			continue
		}
		asset := currency.FormatAsset(p.currencies, symbol)
		if asset != targetAsset {
			continue
		}

		// Remove sign for amount — PSPPayment.Amount is always positive.
		cleanVal := strings.TrimPrefix(val, "-")
		amount, err := currency.GetAmountWithPrecisionFromString(cleanVal, precision)
		if err != nil {
			return nil, err
		}
		return amount, nil
	}
	return big.NewInt(0), nil
}

func buildTransactionMetadata(tx client.UserTransaction) map[string]string {
	metadata := make(map[string]string)
	metadata["type"] = tx.Type

	if tx.Fee != "" && tx.Fee != "0" && tx.Fee != "0.00" {
		metadata["fee"] = tx.Fee
	}
	if tx.OrderID != 0 {
		metadata["order_id"] = strconv.FormatInt(tx.OrderID, 10)
	}

	// Record all currency amounts in metadata for traceability.
	for key, val := range tx.CurrencyAmounts {
		if val != "0" && val != "0.00" {
			metadata["amount_"+key] = val
		}
	}

	return metadata
}

func transactionTypeToPaymentType(txType string) models.PaymentType {
	switch txType {
	case txTypeDeposit:
		return models.PAYMENT_TYPE_PAYIN
	case txTypeWithdrawal:
		return models.PAYMENT_TYPE_PAYOUT
	case txTypeMarketTrade:
		return models.PAYMENT_TYPE_OTHER
	case txTypeSubAccountTransfer, txTypeSettlementTransfer, txTypeInterAccountTransfer:
		return models.PAYMENT_TYPE_TRANSFER
	case txTypeStakingCredit, txTypeStakingSent, txTypeStakingReward:
		return models.PAYMENT_TYPE_OTHER
	case txTypeReferralReward:
		return models.PAYMENT_TYPE_OTHER
	default:
		return models.PAYMENT_TYPE_OTHER
	}
}
