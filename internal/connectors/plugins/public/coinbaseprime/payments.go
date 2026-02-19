package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/coinbaseprime/client"
	"github.com/formancehq/payments/internal/models"
)

type paymentsState struct {
	Cursor string `json:"cursor"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	response, err := p.client.GetTransactions(ctx, oldState.Cursor, req.PageSize)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	payments := make([]models.PSPPayment, 0, len(response.Transactions))
	for _, tx := range response.Transactions {
		payment, err := transactionToPayment(tx)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to convert transaction %s: %w", tx.ID, err)
		}
		if payment == nil {
			continue
		}
		payments = append(payments, *payment)
	}

	newState := paymentsState{Cursor: response.Pagination.NextCursor}
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  response.Pagination.HasNext,
	}, nil
}

func transactionToPayment(tx client.Transaction) (*models.PSPPayment, error) {
	asset, precision, ok := resolveAssetAndPrecision(tx.Symbol)
	if !ok {
		return nil, nil
	}

	raw, err := json.Marshal(tx)
	if err != nil {
		return nil, err
	}

	paymentType := transactionTypeToPaymentType(tx.Type)
	status := transactionStatusToPaymentStatus(tx.Status)

	amount, err := currency.GetAmountWithPrecisionFromString(tx.Amount, precision)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount: %w", err)
	}

	metadata := make(map[string]string)

	payment := models.PSPPayment{
		Reference: tx.ID,
		CreatedAt: tx.CreatedAt,
		Type:      paymentType,
		Amount:    amount,
		Asset:     asset,
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    status,
		Metadata:  metadata,
		Raw:       raw,
	}

	sourceAccountReference, destinationAccountReference := resolveAccountReferences(tx)
	payment.SourceAccountReference = sourceAccountReference
	payment.DestinationAccountReference = destinationAccountReference

	return &payment, nil
}

func resolveAssetAndPrecision(symbol string) (string, int, bool) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	precision, err := currency.GetPrecision(supportedCurrenciesWithDecimal, symbol)
	if err != nil {
		return "", 0, false
	}

	asset := currency.FormatAsset(supportedCurrenciesWithDecimal, symbol)
	return asset, precision, true
}

func resolveAccountReferences(tx client.Transaction) (*string, *string) {
	var source, dest *string

	// Use transfer_from/transfer_to when available (all tx types can have them)
	if tx.TransferFrom != nil && tx.TransferFrom.Value != "" {
		v := tx.TransferFrom.Value
		source = &v
	}
	if tx.TransferTo != nil && tx.TransferTo.Value != "" {
		v := tx.TransferTo.Value
		dest = &v
	}

	// Fallback to walletID based on payment direction
	if tx.WalletID != "" {
		switch transactionTypeToPaymentType(tx.Type) {
		case models.PAYMENT_TYPE_PAYIN:
			if dest == nil {
				v := tx.WalletID
				dest = &v
			}
		case models.PAYMENT_TYPE_PAYOUT:
			if source == nil {
				v := tx.WalletID
				source = &v
			}
		}
	}

	return source, dest
}

func transactionTypeToPaymentType(txType string) models.PaymentType {
	switch strings.ToUpper(txType) {
	case "DEPOSIT", "SWEEP_DEPOSIT", "PROXY_DEPOSIT",
		"COINBASE_DEPOSIT", "COINBASE_REFUND", "REWARD",
		"DEPOSIT_ADJUSTMENT", "CLAIM_REWARDS":
		return models.PAYMENT_TYPE_PAYIN
	case "WITHDRAWAL", "SWEEP_WITHDRAWAL",
		"PROXY_WITHDRAWAL", "BILLING_WITHDRAWAL", "WITHDRAWAL_ADJUSTMENT":
		return models.PAYMENT_TYPE_PAYOUT
	case "CONVERSION", "INTERNAL_DEPOSIT", "INTERNAL_WITHDRAWAL":
		return models.PAYMENT_TYPE_TRANSFER
	default:
		return models.PAYMENT_TYPE_OTHER
	}
}

func transactionStatusToPaymentStatus(status string) models.PaymentStatus {
	switch strings.ToUpper(status) {
	case "TRANSACTION_PENDING", "TRANSACTION_CREATED", "TRANSACTION_REQUESTED",
		"TRANSACTION_APPROVED", "TRANSACTION_GASSING", "TRANSACTION_GASSED",
		"TRANSACTION_PROVISIONED", "TRANSACTION_PLANNED", "TRANSACTION_PROCESSING",
		"TRANSACTION_RESTORED", "TRANSACTION_IMPORT_PENDING", "TRANSACTION_DELAYED",
		"TRANSACTION_BROADCASTING", "TRANSACTION_CONSTRUCTED":
		return models.PAYMENT_STATUS_PENDING
	case "TRANSACTION_DONE", "TRANSACTION_IMPORTED":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "TRANSACTION_CANCELLED":
		return models.PAYMENT_STATUS_CANCELLED
	case "TRANSACTION_EXPIRED":
		return models.PAYMENT_STATUS_EXPIRED
	case "TRANSACTION_FAILED", "TRANSACTION_REJECTED":
		return models.PAYMENT_STATUS_FAILED
	case "OTHER_TRANSACTION_STATUS":
		return models.PAYMENT_STATUS_OTHER
	default:
		return models.PAYMENT_STATUS_UNKNOWN
	}
}
