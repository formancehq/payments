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

	response, err := p.client.GetTransfers(ctx, oldState.Cursor, req.PageSize)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	payments := make([]models.PSPPayment, 0, len(response.Transfers))
	for _, transfer := range response.Transfers {
		payment, err := transferToPayment(transfer)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to convert transfer %s: %w", transfer.ID, err)
		}
		if payment == nil {
			// Skip unsupported currencies
			continue
		}
		payments = append(payments, *payment)
	}

	newState := paymentsState{Cursor: response.NextCursor}
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  response.HasMore,
	}, nil
}

func transferToPayment(transfer client.Transfer) (*models.PSPPayment, error) {
	precision, ok := supportedCurrenciesWithDecimal[transfer.Currency]
	if !ok {
		// Skip unsupported currencies
		return nil, nil
	}

	raw, err := json.Marshal(transfer)
	if err != nil {
		return nil, err
	}

	paymentType := transferTypeToPaymentType(transfer.Type)
	status := transferToStatus(transfer)

	// Remove negative sign if present (for withdrawals)
	amountStr := strings.TrimPrefix(transfer.Amount, "-")

	amount, err := currency.GetAmountWithPrecisionFromString(amountStr, precision)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount: %w", err)
	}

	asset := currency.FormatAsset(supportedCurrenciesWithDecimal, transfer.Currency)

	metadata := map[string]string{
		"transfer_type": transfer.Type,
	}

	// Add debugging fields for transfer details
	if transfer.Details.CoinbaseAccountID != "" {
		metadata["coinbase_account_id"] = transfer.Details.CoinbaseAccountID
	}
	if transfer.Details.CoinbaseTransactionID != "" {
		metadata["coinbase_transaction_id"] = transfer.Details.CoinbaseTransactionID
	}
	if transfer.Details.CoinbasePaymentMethodID != "" {
		metadata["coinbase_payment_method_id"] = transfer.Details.CoinbasePaymentMethodID
	}
	if transfer.Details.CryptoTransactionHash != "" {
		metadata["crypto_transaction_hash"] = transfer.Details.CryptoTransactionHash
	}
	if transfer.Details.CryptoAddress != "" {
		metadata["crypto_address"] = transfer.Details.CryptoAddress
	}
	if transfer.Details.DestinationTag != "" {
		metadata["destination_tag"] = transfer.Details.DestinationTag
	}
	if transfer.Details.SentToAddress != "" {
		metadata["sent_to_address"] = transfer.Details.SentToAddress
	}

	payment := models.PSPPayment{
		Reference: transfer.ID,
		CreatedAt: transfer.CreatedAt,
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

func transferTypeToPaymentType(transferType string) models.PaymentType {
	switch strings.ToLower(transferType) {
	case "deposit", "internal_deposit":
		return models.PAYMENT_TYPE_PAYIN
	case "withdraw", "internal_withdraw":
		return models.PAYMENT_TYPE_PAYOUT
	default:
		return models.PAYMENT_TYPE_TRANSFER
	}
}

func transferToStatus(transfer client.Transfer) models.PaymentStatus {
	if transfer.CanceledAt != nil {
		return models.PAYMENT_STATUS_CANCELLED
	}
	if transfer.CompletedAt != nil {
		return models.PAYMENT_STATUS_SUCCEEDED
	}
	if transfer.ProcessedAt != nil {
		return models.PAYMENT_STATUS_PENDING
	}
	return models.PAYMENT_STATUS_PENDING
}
