package coinbase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/coinbase/client"
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

	metadata := map[string]string{
		"transaction_type": tx.Type,
	}

	if tx.WalletID != "" {
		metadata["wallet_id"] = tx.WalletID
	}
	if tx.PortfolioID != "" {
		metadata["portfolio_id"] = tx.PortfolioID
	}
	if tx.TransferFrom != "" {
		metadata["transfer_from"] = tx.TransferFrom
	}
	if tx.TransferTo != "" {
		metadata["transfer_to"] = tx.TransferTo
	}
	if tx.Network != "" {
		metadata["network"] = tx.Network
	}
	if tx.Fees != "" {
		metadata["fees"] = tx.Fees
	}
	if tx.FeeSymbol != "" {
		metadata["fee_symbol"] = tx.FeeSymbol
	}
	if tx.NetworkFees != "" {
		metadata["network_fees"] = tx.NetworkFees
	}
	if len(tx.BlockchainIDs) > 0 {
		metadata["blockchain_ids"] = strings.Join(tx.BlockchainIDs, ",")
	}

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
	precision, err := currency.GetPrecision(supportedCurrenciesWithDecimal, symbol)
	if err != nil {
		return "", 0, false
	}

	asset := currency.FormatAsset(supportedCurrenciesWithDecimal, symbol)
	return asset, precision, true
}

func resolveAccountReferences(tx client.Transaction) (*string, *string) {
	var sourceAccountReference *string
	var destinationAccountReference *string

	switch strings.ToUpper(tx.Type) {
	case "DEPOSIT":
		if tx.WalletID != "" {
			walletID := tx.WalletID
			destinationAccountReference = &walletID
		}
	case "WITHDRAWAL":
		if tx.WalletID != "" {
			walletID := tx.WalletID
			sourceAccountReference = &walletID
		}
	case "INTERNAL_TRANSFER":
		if tx.TransferFrom != "" {
			transferFrom := tx.TransferFrom
			sourceAccountReference = &transferFrom
		}
		if tx.TransferTo != "" {
			transferTo := tx.TransferTo
			destinationAccountReference = &transferTo
		}
	}

	return sourceAccountReference, destinationAccountReference
}

func transactionTypeToPaymentType(txType string) models.PaymentType {
	switch strings.ToUpper(txType) {
	case "DEPOSIT":
		return models.PAYMENT_TYPE_PAYIN
	case "WITHDRAWAL":
		return models.PAYMENT_TYPE_PAYOUT
	case "INTERNAL_TRANSFER":
		return models.PAYMENT_TYPE_TRANSFER
	default:
		return models.PAYMENT_TYPE_TRANSFER
	}
}

func transactionStatusToPaymentStatus(status string) models.PaymentStatus {
	switch strings.ToUpper(status) {
	case "TRANSACTION_COMPLETED":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "TRANSACTION_FAILED":
		return models.PAYMENT_STATUS_FAILED
	default:
		return models.PAYMENT_STATUS_PENDING
	}
}
