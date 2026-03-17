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
	txTypeDeposit              = "0"
	txTypeWithdrawal           = "1"
	txTypeMarketTrade          = "2"
	txTypeSubAccountTransfer   = "14"
	txTypeStakingCredit        = "25"
	txTypeStakingSent          = "26"
	txTypeStakingReward        = "27"
	txTypeReferralReward       = "32"
	txTypeSettlementTransfer   = "33"
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
	asset, amount, ok, err := p.resolveAssetAndAmount(tx.CurrencyAmounts)
	if err != nil {
		return nil, err
	}
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

// resolveAssetAndAmount finds the primary currency in the transaction using a
// deterministic strategy: pick the currency with the largest absolute value
// among known currencies. Returns the formatted asset, the parsed amount
// (always positive), and whether a match was found. This is a single-pass
// replacement for the former resolveAssetAndPrecision + extractAmount.
func (p *Plugin) resolveAssetAndAmount(amounts map[string]string) (string, *big.Int, bool, error) {
	var bestSymbol string
	var bestPrecision int
	var bestRawVal string
	var bestAbsVal float64
	found := false

	for key, val := range amounts {
		symbol := normalizeCurrency(key)
		precision, ok := p.currencies[symbol]
		if !ok {
			continue
		}

		cleanVal := strings.TrimPrefix(val, "-")
		if isZeroAmount(cleanVal) {
			continue
		}

		fval, _, err := new(big.Float).Parse(cleanVal, 10)
		if err != nil {
			continue
		}
		absVal, _ := fval.Float64()

		if !found || absVal > bestAbsVal {
			bestSymbol = symbol
			bestPrecision = precision
			bestRawVal = cleanVal
			bestAbsVal = absVal
			found = true
		}
	}

	if !found {
		return "", nil, false, nil
	}

	asset := currency.FormatAsset(p.currencies, bestSymbol)
	amount, err := currency.GetAmountWithPrecisionFromString(bestRawVal, bestPrecision)
	if err != nil {
		return "", nil, false, fmt.Errorf("failed to parse amount for %s: %w", bestSymbol, err)
	}

	return asset, amount, true, nil
}

func buildTransactionMetadata(tx client.UserTransaction) map[string]string {
	metadata := make(map[string]string)
	metadata["type"] = tx.Type

	if !isZeroAmount(tx.Fee) {
		metadata["fee"] = tx.Fee
	}
	if tx.OrderID != 0 {
		metadata["order_id"] = strconv.FormatInt(tx.OrderID, 10)
	}

	// Record all currency amounts in metadata for traceability.
	for key, val := range tx.CurrencyAmounts {
		if !isZeroAmount(val) {
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
