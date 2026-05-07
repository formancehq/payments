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
	// Bitstamp pagination is based on monotonically increasing transaction IDs.
	// Historical transaction updates are not revisited by since_id; Bitstamp's
	// user_transactions endpoint is treated here as settled transaction history.
	LastTransactionID int64 `json:"lastTransactionID"`
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
	txTypeBuySell              = "36"
)

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	currencies, err := p.getCurrencies(ctx)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	var sinceID *int64
	if oldState.LastTransactionID > 0 {
		sinceID = &oldState.LastTransactionID
	}
	transactions, err := p.client.GetUserTransactions(ctx, sinceID, req.PageSize)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	payments := make([]models.PSPPayment, 0, len(transactions))
	lastTransactionID := oldState.LastTransactionID
	for _, tx := range transactions {
		if tx.ID > lastTransactionID {
			lastTransactionID = tx.ID
		}

		payment, err := p.transactionToPayment(tx, currencies)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to convert transaction %d: %w", tx.ID, err)
		}
		if payment == nil {
			continue
		}
		payments = append(payments, *payment)
	}

	newState := paymentsState{LastTransactionID: lastTransactionID}
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

func (p *Plugin) transactionToPayment(tx client.UserTransaction, currencies map[string]int) (*models.PSPPayment, error) {
	if isOrderTransaction(tx) {
		p.logger.Infof("skipping transaction %d: order transaction type %s", tx.ID, tx.Type)
		return nil, nil
	}

	asset, amount, ok, err := resolveAssetAndAmount(currencies, tx.CurrencyAmounts)
	if err != nil {
		return nil, err
	}
	if !ok {
		p.logger.Infof("skipping transaction %d: expected exactly one matching currency amount", tx.ID)
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

// resolveAssetAndAmount accepts only payment-like transactions with exactly
// one non-zero known currency amount. Orders/conversions expose multiple assets
// and are filtered before this point.
func resolveAssetAndAmount(currencies map[string]int, amounts map[string]string) (string, *big.Int, bool, error) {
	var selectedSymbol string
	var selectedPrecision int
	var selectedRawVal string
	count := 0

	for key, val := range amounts {
		symbol := normalizeCurrency(key)
		precision, ok := currencies[symbol]
		if !ok {
			continue
		}

		cleanVal := strings.TrimPrefix(val, "-")
		if isZeroAmount(cleanVal) {
			continue
		}

		count++
		if count > 1 {
			return "", nil, false, nil
		}
		selectedSymbol = symbol
		selectedPrecision = precision
		selectedRawVal = cleanVal
	}

	if count == 0 {
		return "", nil, false, nil
	}

	asset := currency.FormatAsset(currencies, selectedSymbol)
	amount, err := currency.GetAmountWithPrecisionFromString(selectedRawVal, selectedPrecision)
	if err != nil {
		return "", nil, false, fmt.Errorf("failed to parse amount for %s: %w", selectedSymbol, err)
	}

	return asset, amount, true, nil
}

func buildTransactionMetadata(tx client.UserTransaction) map[string]string {
	metadata := make(map[string]string)
	metadata[MetadataPrefix+"type"] = tx.Type

	if !isZeroAmount(tx.Fee) {
		metadata[MetadataPrefix+"fee"] = tx.Fee
	}
	if tx.OrderID != 0 {
		metadata[MetadataPrefix+"order_id"] = strconv.FormatInt(tx.OrderID, 10)
	}
	if tx.Market != "" {
		metadata[MetadataPrefix+"market"] = tx.Market
	}

	return metadata
}

func isOrderTransaction(tx client.UserTransaction) bool {
	return tx.Type == txTypeMarketTrade || tx.Type == txTypeBuySell || tx.Market != ""
}

func transactionTypeToPaymentType(txType string) models.PaymentType {
	switch txType {
	case txTypeDeposit:
		return models.PAYMENT_TYPE_PAYIN
	case txTypeWithdrawal:
		return models.PAYMENT_TYPE_PAYOUT
	case txTypeSubAccountTransfer, txTypeSettlementTransfer, txTypeInterAccountTransfer:
		return models.PAYMENT_TYPE_TRANSFER
	case txTypeStakingReward, txTypeReferralReward:
		return models.PAYMENT_TYPE_PAYIN
	case txTypeStakingCredit, txTypeStakingSent:
		return models.PAYMENT_TYPE_OTHER
	default:
		return models.PAYMENT_TYPE_OTHER
	}
}
